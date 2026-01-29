package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/cdc/event"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// CDC 幂等操作 redis 缓存过期时间。
	cdcIdempotentCacheTTL = 24 * time.Hour
)

// CDCService 是 CDC 服务接口。
type CDCService interface {
	// ProcessContact 处理联系人申请审批事件。
	// 审批通过建立联系人关联关系。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessContact(ctx context.Context, msg *xmq.Message) error

	// ProcessConversation 处理会话创建事件。
	// 用户联系人关系建立后，创建会话。
	ProcessConversation(ctx context.Context, msg *xmq.Message) error
}

var _ CDCService = (*DefaultCDCService)(nil)

type DefaultCDCService struct {
	rdb              redis.Cmdable
	contactRepo      repo.ContactRepo
	conversationRepo repo.ConversationRepo

	logger *zap.Logger
}

func NewDefaultCDCService(
	rdb redis.Cmdable,
	userContactRepo repo.ContactRepo,
	logger *zap.Logger,
) *DefaultCDCService {
	return &DefaultCDCService{
		rdb:         rdb,
		contactRepo: userContactRepo,

		logger: logger,
	}
}

func (s *DefaultCDCService) ProcessContact(ctx context.Context, msg *xmq.Message) error {
	var event event.ContactApplicationCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal contact application cdc event: %w", err)
	}

	if !event.IsApproved() {
		return nil
	}

	// 幂等性判断 ( 防止重复处理相同的联系人申请事件 )。
	cacheKey := fmt.Sprintf("contact:application:%d", event.ID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug("[cdc-contact] contact application already processed", zap.Uint64("application_id", event.ID))
		return nil
	}

	// 建立双向关联关系。
	eg, ctx := errgroup.WithContext(ctx)

	// 创建 A -> B 的联系人关系。
	eg.Go(func() error {
		return s.createContact(ctx, domain.Contact{
			UserID:    event.ApplicantID,
			ContactID: event.TargetID,
			Source:    event.Source,
			AddedAt:   event.CreatedAt,
		})
	})

	// 创建 B -> A 的联系人关系。
	eg.Go(func() error {
		return s.createContact(ctx, domain.Contact{
			UserID:    event.TargetID,
			ContactID: event.ApplicantID,
			Source:    event.Source,
			AddedAt:   event.ReviewedAt,
		})
	})

	if err := eg.Wait(); err != nil {
		s.logger.Error("[cdc-contact] failed to create contact relation", zap.Uint64("application_id", event.ID), zap.Error(err))
		return err
	}

	// 关联关系建立成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

// createContact 创建联系人关系。
// 如果联系人关系已经存在但被删除了，则重新添加。
// 如果联系人关系不存在，则创建。
func (s *DefaultCDCService) createContact(ctx context.Context, contact domain.Contact) error {
	// 查询联系人关系。
	contact, err := s.contactRepo.FindByUserIDAndContactID(ctx, contact.UserID, contact.ContactID)
	if err != nil {
		return err
	}

	if contact.ID != 0 && contact.DeletedAt != 0 {
		// 以前添加过联系人但被删除了，现在重新添加。
		_, err = s.contactRepo.Recreate(ctx, contact)
		if err != nil {
			return err
		}
	}

	_, err = s.contactRepo.Create(ctx, contact)
	if err != nil {
		return err
	}
	return nil
}

// createContactSync 创建联系人关系。
func (s *DefaultCDCService) ProcessConversation(ctx context.Context, msg *xmq.Message) error {
	var event event.ContactCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal user contact cdc event: %w", err)
	}

	if !event.IsUpdate() {
		s.logger.Debug("[cdc-conversation] conversation event is not update", zap.Any("event", event))
		return nil
	}

	if event.DeletedAt != 0 {
		// TODO: cdc 事件为删除用户联系人 ( 逻辑删除 )，删除会话。
		panic("not implemented")
	}

	// cdc 事件为创建联系人关系事件，创建会话。
	return s.createConversation(ctx, domain.ConversationView{
		UserID:           event.UserID,
		TargetID:         event.ContactID,
		ConversationType: domain.ConversationTypeSingle,
		OpenedAt:         event.AddedAt,
	})
}

func (s *DefaultCDCService) createConversation(ctx context.Context, conversation domain.ConversationView) error {
	// 幂等性判断 ( 防止重复处理相同的联系人关系建立事件 )。
	cacheKey := fmt.Sprintf("conversation:relation:%d-%d", conversation.UserID, conversation.TargetID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-conversation] conversation already processed",
			zap.Uint64("user_id", conversation.UserID),
			zap.Uint64("contact_id", conversation.TargetID),
		)
		return nil
	}

	// 创建会话 ( 这里消费的是联系人关系建立事件，只需要创建会话即可 )。
	err := s.conversationRepo.Create(ctx, conversation)
	if err != nil {
		s.logger.Error("[cdc-conversation] failed to create conversation", zap.Error(err))
		return err
	}

	// 关联关系建立成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

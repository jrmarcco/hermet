package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/cdc/event"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
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
	// ProcessContactReview 处理联系人申请审批事件。
	// 审批通过建立联系人关联关系。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessContactReview(ctx context.Context, msg *xmq.Message) error

	// ProcessContact 处理联系人关系建立事件。
	// 用户联系人关系建立后，创建会话。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessContact(ctx context.Context, msg *xmq.Message) error

	// ProcessChannelMember 处理频道成员添加事件。
	// 频道成员添加后，建立成员视图 ( 冗余成员信息 )。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessChannelMember(ctx context.Context, msg *xmq.Message) error
}

var _ CDCService = (*DefaultCDCService)(nil)

type DefaultCDCService struct {
	rdb redis.Cmdable

	userRepo         repo.BizUserRepo
	contactRepo      repo.ContactRepo
	channelRepo      repo.ChannelRepo
	conversationRepo repo.ConversationRepo

	logger *zap.Logger
}

func NewDefaultCDCService(
	rdb redis.Cmdable,
	userRepo repo.BizUserRepo,
	contactRepo repo.ContactRepo,
	channelRepo repo.ChannelRepo,
	conversationRepo repo.ConversationRepo,
	logger *zap.Logger,
) *DefaultCDCService {
	return &DefaultCDCService{
		rdb: rdb,

		userRepo:         userRepo,
		contactRepo:      contactRepo,
		channelRepo:      channelRepo,
		conversationRepo: conversationRepo,

		logger: logger,
	}
}

func (s *DefaultCDCService) ProcessContactReview(ctx context.Context, msg *xmq.Message) error {
	var event event.ContactReviewCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal contact application cdc event: %w", err)
	}

	if !event.IsApproved() {
		// 只处理审批通过事件。
		return nil
	}

	// 幂等性判断 ( 防止重复处理相同的联系人申请事件 )。
	cacheKey := fmt.Sprintf("contact:review:%d", event.ID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug("[cdc-contact-review] contact review already processed", zap.Uint64("review_id", event.ID))
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
		s.logger.Error(
			"[cdc-contact-review] failed to create contact relation",
			zap.Uint64("review_id", event.ID),
			zap.Error(err),
		)
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

	if _, err = s.contactRepo.Create(ctx, contact); err != nil {
		return err
	}
	return nil
}

// createContactSync 创建联系人关系。
func (s *DefaultCDCService) ProcessContact(ctx context.Context, msg *xmq.Message) error {
	var event event.ContactCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal user contact cdc event: %w", err)
	}

	if !(event.IsCreate() || event.IsUpdate()) {
		// 只处理 create 和 update 事件。
		// - create: 创建联系人关系事件。
		// - update: 联系人被删除 ( 逻辑删除 DeletedAt != 0 )
		// 注意: 重新添加需要重新审批，所以不会触发 update 事件 ( LeftAt != 0 -> LeftAt = 0 )。
		return nil
	}

	if event.IsUpdate() && event.DeletedAt != 0 {
		// TODO: cdc 事件为删除用户联系人 ( 逻辑删除 )。
		return nil
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
	cacheKey := fmt.Sprintf("contact:create:%d-%d", conversation.UserID, conversation.TargetID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-contact] contact already processed",
			zap.Uint64("user_id", conversation.UserID),
			zap.Uint64("contact_id", conversation.TargetID),
		)
		return nil
	}

	// 创建会话 ( 这里消费的是联系人关系建立事件，只需要创建会话即可 )。
	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		s.logger.Error("[cdc-contact] failed to create contact", zap.Error(err))
		return err
	}

	// 关联关系建立成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

func (s *DefaultCDCService) ProcessChannelMember(ctx context.Context, msg *xmq.Message) error {
	var event event.ChannelMemberCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal channel member cdc event: %w", err)
	}

	if !(event.IsCreate() || event.IsUpdate()) {
		// 只处理 create 和 update 事件。
		// - create: 频道成员添加事件。
		// - update:
		// 	 1. 频道成员离开 ( 逻辑删除 LeftAt != 0 )。
		//   2. 频道成员重新加入 ( LeftAt != 0 -> LeftAt = 0 )。
		return nil
	}

	if event.IsUpdate() && event.LeftAt != 0 {
		// TODO: cdc 事件为频道成员离开事件 ( 逻辑删除 )。
		return nil
	}

	// cdc 事件为频道成员添加事件，创建频道成员视图。
	user, err := s.userRepo.FindByID(ctx, event.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			s.logger.Error("[cdc-channel-member] user not found", zap.Uint64("user_id", event.UserID), zap.Error(err))
			return err
		}
		return fmt.Errorf("failed to find user: %w", err)
	}
	cmv := domain.ChannelMemberView{
		ID: event.ID,

		CID: event.ChannelID,
		UID: event.UserID,

		MemberRole: domain.ChannelMemberRole(event.MemberRole),

		MemberNickname: user.Nickname,
		MemberAvatar:   user.Avatar,
		MemberGender:   user.Gender,
		MemberRemark:   user.Nickname,
		MemberInfoVer:  user.InfoVer,

		JoinedAt: event.JoinedAt,
		LeftAt:   event.LeftAt,
	}
	return s.createCmv(ctx, cmv)
}

func (s *DefaultCDCService) createCmv(ctx context.Context, cmv domain.ChannelMemberView) error {
	// 幂等性判断 ( 防止重复处理相同的频道成员添加事件 )。
	cacheKey := fmt.Sprintf("channel:member:create:%d-%d", cmv.CID, cmv.UID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-channel-member] channel member already processed",
			zap.Uint64("channel_id", cmv.CID),
			zap.Uint64("user_id", cmv.UID),
		)
		return nil
	}

	if _, err := s.channelRepo.CreateMemberView(ctx, cmv); err != nil {
		return fmt.Errorf("failed to create channel member view: %w", err)
	}

	// 频道成员视图创建成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

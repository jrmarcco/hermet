package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	contactApplicationCacheTTL = 24 * time.Hour
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
	rdb             redis.Cmdable
	userContactRepo repo.UserContactRepo

	logger *zap.Logger
}

func NewDefaultCDCService(
	rdb redis.Cmdable,
	userContactRepo repo.UserContactRepo,
	logger *zap.Logger,
) *DefaultCDCService {
	return &DefaultCDCService{
		rdb:             rdb,
		userContactRepo: userContactRepo,

		logger: logger,
	}
}

func (s *DefaultCDCService) ProcessContact(ctx context.Context, msg *xmq.Message) error {
	var event domain.ContactApplicationCDCEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal contact application cdc event: %w", err)
	}

	if !event.IsApproved() {
		return nil
	}

	// 幂等性判断 ( 防止重复处理相同的联系人申请事件 )。
	cacheKey := fmt.Sprintf("contact:application:%d", event.ID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug("[hermet-cdc-contact] contact application already processed", zap.Uint64("application_id", event.ID))
		return nil
	}

	// 建立双向关联关系。
	eg, ctx := errgroup.WithContext(ctx)

	// 创建 A -> B 的联系人关系。
	eg.Go(func() error {
		_, err := s.userContactRepo.Create(ctx, domain.UserContact{
			UserID:    event.ApplicantID,
			ContactID: event.TargetID,
			Source:    event.Source,
			AddedAt:   event.CreatedAt,
		})
		return err
	})

	// 创建 B -> A 的联系人关系。
	eg.Go(func() error {
		_, err := s.userContactRepo.Create(ctx, domain.UserContact{
			UserID:    event.TargetID,
			ContactID: event.ApplicantID,
			Source:    event.Source,
			AddedAt:   event.CreatedAt,
		})
		return err
	})

	if err := eg.Wait(); err != nil {
		s.logger.Error("[hermet-cdc-contact] failed to create contact relation", zap.Uint64("application_id", event.ID), zap.Error(err))
		return err
	}

	// 关联关系建立成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", contactApplicationCacheTTL)
	return nil
}

func (s *DefaultCDCService) ProcessConversation(_ context.Context, _ *xmq.Message) error {
	// TODO: not implemented.
	panic("not implemented")
}

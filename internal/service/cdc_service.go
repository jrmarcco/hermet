package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type CDCService interface {
	// ProcessContact 处理联系人申请审批事件。
	// 审批通过建立联系人关联关系。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessContact(ctx context.Context, msg *xmq.Message) error
}

type DefaultCDCService struct {
	userContactRepo repo.UserContactRepo

	logger *zap.Logger
}

func NewDefaultCDCService(
	userContactRepo repo.UserContactRepo,
	logger *zap.Logger,
) *DefaultCDCService {
	return &DefaultCDCService{
		userContactRepo: userContactRepo,
		logger:          logger,
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

	// TODO: 判断联系人关系是否已经建立。

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

	return eg.Wait()
}

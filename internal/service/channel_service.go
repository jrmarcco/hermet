package service

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
)

type ChannelService interface {
	Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error)
}

var _ ChannelService = (*DefaultChannelService)(nil)

type DefaultChannelService struct {
	asyncThreshold int // 同步成员写入的阈值

	channelRepo repo.ChannelRepo
	logger      *zap.Logger
}

func NewDefaultChannelService(channelRepo repo.ChannelRepo, logger *zap.Logger) *DefaultChannelService {
	return &DefaultChannelService{
		channelRepo: channelRepo,
		logger:      logger,
	}
}

func (s *DefaultChannelService) Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error) {
	memberCnt := len(event.MemberIDs)
	if memberCnt == 0 {
		return domain.Channel{}, fmt.Errorf("%w: member is empty", errs.ErrInvalidParam)
	}

	// 根据成员数量选择创建策略。
	if memberCnt <= s.asyncThreshold {
		s.logger.Debug(
			"[hermet-channel] create channel sync",
			zap.String("name", event.Name),
			zap.Int("member_count", memberCnt),
		)
		return s.createSync(ctx, event)
	}
	return s.createAsync(ctx, event)
}

// createSync 同步创建群组 ( 适用于小群组 )。
func (s *DefaultChannelService) createSync(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error) {
	channel := domain.Channel{
		ChannelName:   event.Name,
		ChannelAvatar: event.Avatar,
		ChannelStatus: domain.ChannelStatusActive,
		CreatorID:     event.Creator,
	}

	_, err := s.channelRepo.CreateWithMembers(ctx, channel, event.MemberIDs)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("failed to create channel with members: %w", err)
	}
	return channel, nil
}

// createAsync 异步创建群组 ( 适用于大群组 )。
func (s *DefaultChannelService) createAsync(_ context.Context, _ domain.ChannelCreateEvent) (domain.Channel, error) {
	// TODO: not implemented
	panic("not implemented")
}

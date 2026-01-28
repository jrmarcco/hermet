package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
)

type ChannelService interface {
	Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error)
}

var _ ChannelService = (*DefaultChannelService)(nil)

type DefaultChannelService struct {
	asyncThreshold     int    // 同步成员写入的阈值
	memberSyncAddTopic string // 成员同步添加主题

	channelRepo repo.ChannelRepo
	producer    xproduce.Producer

	logger *zap.Logger
}

func NewDefaultChannelService(
	asyncThreshold int,
	memberSyncAddTopic string,
	channelRepo repo.ChannelRepo,
	producer xproduce.Producer,
	logger *zap.Logger,
) *DefaultChannelService {
	return &DefaultChannelService{
		asyncThreshold:     asyncThreshold,
		memberSyncAddTopic: memberSyncAddTopic,

		channelRepo: channelRepo,
		producer:    producer,

		logger: logger,
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
func (s *DefaultChannelService) createAsync(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error) {
	// 1. 创建 channel ( 状态为 "creating" )。
	channel := domain.Channel{
		ChannelName:   event.Name,
		ChannelAvatar: event.Avatar,
		ChannelStatus: domain.ChannelStatusCreating,
		CreatorID:     event.Creator,
	}
	_, err := s.channelRepo.Create(ctx, channel)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("failed to create channel: %w", err)
	}

	// 2. 批量添加成员。
	// 分批发送 ( 每批 500 个成员 )。
	const batchSize = 500
	memberCount := len(event.MemberIDs)
	for i := 0; i < memberCount; i += batchSize {
		end := min(i+batchSize, memberCount)
		batch := event.MemberIDs[i:end]

		msg := domain.ChannelMemberAddEvent{
			CID:       channel.ID,
			MemberIDs: batch,
			Operator:  event.Creator,
		}

		val, err := json.Marshal(msg)
		if err != nil {
			return domain.Channel{}, fmt.Errorf("failed to marshal channel member add event: %w", err)
		}

		err = s.producer.Produce(ctx, &xmq.Message{
			Topic: s.memberSyncAddTopic,
			// 使用 Channel 的 ID 作为 key 确保成员发送到同一个 kafka 分区。
			Key: []byte(strconv.FormatUint(channel.ID, 10)),
			Val: val,
		})
		if err != nil {
			// 发送失败，更新频道状态为 "failed"。
			_ = s.channelRepo.UpdateStatus(ctx, channel.ID, string(domain.ChannelStatusFailed))
			return domain.Channel{}, fmt.Errorf("failed to produce channel member add event: %w", err)
		}
	}
	return channel, nil
}

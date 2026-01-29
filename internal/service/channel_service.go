package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
)

// ChannelMsgProducer 是频道消息生产者接口。
type ChannelMsgProducer interface {
	Produce(ctx context.Context, key []byte, val []byte) error
}

type ChannelService interface {
	Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error)
	AddMembers(ctx context.Context, event domain.ChannelMemberAddEvent) error

	ProcessAddMembersMsg(ctx context.Context, msg *xmq.Message) error
}

var _ ChannelService = (*DefaultChannelService)(nil)

type DefaultChannelService struct {
	msgProducer ChannelMsgProducer
	channelRepo repo.ChannelRepo

	logger *zap.Logger
}

func NewDefaultChannelService(
	msgProducer ChannelMsgProducer,
	channelRepo repo.ChannelRepo,
	logger *zap.Logger,
) (*DefaultChannelService, error) {
	return &DefaultChannelService{
		msgProducer: msgProducer,
		channelRepo: channelRepo,

		logger: logger,
	}, nil
}

func (s *DefaultChannelService) Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error) {
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

	// 2. 批量添加成员 ( 异步 )。
	// 使用 Kafka 消息队列实现异步。
	if err := s.sendMemberAddMsg(ctx, channel.ID, event.MemberIDs); err != nil {
		return domain.Channel{}, fmt.Errorf("failed to send member add message: %w", err)
	}
	return channel, nil
}

func (s *DefaultChannelService) AddMembers(ctx context.Context, event domain.ChannelMemberAddEvent) error {
	return s.sendMemberAddMsg(ctx, event.CID, event.MemberIDs)
}

// sendMemberAddMsg 发送成员添加消息。
func (s *DefaultChannelService) sendMemberAddMsg(ctx context.Context, channelID uint64, memberIDs []uint64) error {
	// 分批发送 ( 每批 500 个成员 )。
	const batchSize = 500
	memberCount := len(memberIDs)
	for i := 0; i < memberCount; i += batchSize {
		end := min(i+batchSize, memberCount)
		batch := memberIDs[i:end]

		// 判断是否是最后一批成员。
		isLastBatch := end >= memberCount

		msg := domain.ChannelMemberAddEvent{
			CID:         channelID,
			MemberIDs:   batch,
			IsLastBatch: isLastBatch,
		}

		val, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal channel member add event: %w", err)
		}

		if err := s.msgProducer.Produce(ctx, []byte(strconv.FormatUint(channelID, 10)), val); err != nil {
			// 发送失败，更新频道状态为 "failed"。
			_ = s.channelRepo.UpdateStatus(ctx, channelID, string(domain.ChannelStatusFailed))
			return fmt.Errorf("failed to produce channel member add event: %w", err)
		}
	}
	return nil
}

func (s *DefaultChannelService) ProcessAddMembersMsg(ctx context.Context, msg *xmq.Message) error {
	event := domain.ChannelMemberAddEvent{}
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal channel member add event: %w", err)
	}

	if err := s.addMembers(ctx, event.CID, event.MemberIDs); err != nil {
		// TODO: 需要增加成员添加失败处理。
		// 1. 指数退避重试 3 次 ( 1s -> 2s -> 4s )。
		// 2. 如果 3 次重试都失败，转发消息到死信队列 ( DLQ )。
	}

	if event.IsLastBatch {
		if err := s.channelRepo.UpdateStatus(ctx, event.CID, string(domain.ChannelStatusActive)); err != nil {
			s.logger.Error(
				"[channel-service] failed to update channel status to active",
				zap.Uint64("channel_id", event.CID),
				zap.Error(err),
			)
		}
		s.logger.Debug(
			"[channel-service] channel activated successfully",
			zap.Uint64("channel_id", event.CID),
		)
	}
	return nil
}

// addMembers 实际写入成员到数据库。
func (s *DefaultChannelService) addMembers(ctx context.Context, channelID uint64, memberIDs []uint64) error {
	if err := s.channelRepo.AddMembers(ctx, channelID, memberIDs); err != nil {
		return fmt.Errorf("failed to add members: %w", err)
	}
	return nil
}

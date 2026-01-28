package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
)

type ChannelService interface {
	Create(ctx context.Context, event domain.ChannelCreateEvent) (domain.Channel, error)

	AddMembers(ctx context.Context, event domain.ChannelMemberAddEvent) error
}

var _ ChannelService = (*DefaultChannelService)(nil)

type DefaultChannelService struct {
	memberSyncAddTopic string // 成员同步添加主题

	channelRepo repo.ChannelRepo
	producer    xproduce.Producer

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

func NewDefaultChannelService(
	consumerFactory xconsumer.ConsumerFactory,
	memberSyncAddTopic string,
	memberSyncAddGroupID string,
	memberSyncAddPartition int,
	channelRepo repo.ChannelRepo,
	producer xproduce.Producer,
	logger *zap.Logger,
) (*DefaultChannelService, error) {
	if memberSyncAddTopic == "" {
		return nil, fmt.Errorf("member sync add topic is required")
	}
	if memberSyncAddGroupID == "" {
		return nil, fmt.Errorf("member sync add group id is required")
	}
	if memberSyncAddPartition < 1 {
		return nil, fmt.Errorf("member sync add partition must be greater than 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc := &DefaultChannelService{
		memberSyncAddTopic: memberSyncAddTopic,

		channelRepo: channelRepo,
		producer:    producer,

		ctx:        ctx,
		cancelFunc: cancel,

		logger: logger,
	}

	for i := range memberSyncAddPartition {
		partition := i

		consumer, err := consumerFactory.NewConsumer(memberSyncAddTopic, memberSyncAddGroupID)
		if err != nil {
			logger.Error(
				"[hermet-channel] failed to create member sync add consumer",
				zap.String("topic", memberSyncAddTopic),
				zap.String("group_id", memberSyncAddGroupID),
				zap.Int("partition", partition),
				zap.Error(err),
			)
			return nil, err
		}

		msgChan, err := consumer.ConsumeChan(ctx)
		if err != nil {
			logger.Error(
				"[hermet-channel] failed to get message channel",
				zap.String("topic", memberSyncAddTopic),
				zap.String("group_id", memberSyncAddGroupID),
				zap.Int("partition", partition),
				zap.Error(err),
			)
			return nil, err
		}
		go svc.consumeMemberAddMsg(msgChan)
	}
	return svc, nil
}

// consumeMemberAddMsg 消费成员同步添加消息。
func (s *DefaultChannelService) consumeMemberAddMsg(msgChan <-chan *xmq.Message) {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("[hermet-channel-service] member sync add consumer context done")
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}

			event := domain.ChannelMemberAddEvent{}
			if err := json.Unmarshal(msg.Val, &event); err != nil {
				s.logger.Error(
					"[hermet-channel-service] failed to unmarshal channel member add event",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
					zap.Int("partition", msg.Partition),
					zap.String("message", string(msg.Val)),
					zap.Error(err),
				)
				continue
			}
			// 添加成员失败跳过。
			if err := s.addMembers(s.ctx, event.CID, event.MemberIDs); err != nil {
				s.logger.Error(
					"[hermet-channel-service] failed to add members",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
					zap.Int("partition", msg.Partition),
					zap.String("message", string(msg.Val)),
					zap.Error(err),
				)
				continue
			}
		}
	}
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

		msg := domain.ChannelMemberAddEvent{
			CID:       channelID,
			MemberIDs: batch,
		}

		val, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal channel member add event: %w", err)
		}

		err = s.producer.Produce(ctx, &xmq.Message{
			Topic: s.memberSyncAddTopic,
			// 使用 Channel 的 ID 作为 key 确保成员发送到同一个 kafka 分区。
			Key: []byte(strconv.FormatUint(channelID, 10)),
			Val: val,
		})
		if err != nil {
			// 发送失败，更新频道状态为 "failed"。
			_ = s.channelRepo.UpdateStatus(ctx, channelID, string(domain.ChannelStatusFailed))
			return fmt.Errorf("failed to produce channel member add event: %w", err)
		}
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

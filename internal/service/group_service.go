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

// GroupMemberMsgProducer 是频道消息生产者接口。
type GroupMemberMsgProducer interface {
	Produce(ctx context.Context, key []byte, val []byte) error
}

type GroupService interface {
	Create(ctx context.Context, event domain.GroupCreateEvent) (domain.ChatGroup, error)
	AddMembers(ctx context.Context, event domain.GroupMemberAddEvent) error

	ProcessAddMembersMsg(ctx context.Context, msg *xmq.Message) error
}

const (
	// 成员数阈值：小于此值使用同步创建，大于等于此值使用异步创建
	AsyncThreshold = 100
)

var _ GroupService = (*DefaultGroupService)(nil)

type DefaultGroupService struct {
	msgProducer GroupMemberMsgProducer

	groupRepo       repo.GroupRepo
	groupMemberRepo repo.GroupMemberRepo

	logger *zap.Logger
}

func NewDefaultGroupService(
	producer GroupMemberMsgProducer,
	groupRepo repo.GroupRepo,
	groupMemberRepo repo.GroupMemberRepo,
	logger *zap.Logger,
) (*DefaultGroupService, error) {
	return &DefaultGroupService{
		msgProducer: producer,

		groupRepo:       groupRepo,
		groupMemberRepo: groupMemberRepo,

		logger: logger,
	}, nil
}

func (s *DefaultGroupService) Create(ctx context.Context, event domain.GroupCreateEvent) (domain.ChatGroup, error) {
	// 1. 创建聊天群 ( 状态为 "creating" )。
	group, err := s.groupRepo.Create(ctx, domain.ChatGroup{
		GroupName:   event.Name,
		GroupAvatar: event.Avatar,
		GroupStatus: domain.ChatGroupStatusCreating,
		CreatorID:   event.CreatorID,
	})
	if err != nil {
		return domain.ChatGroup{}, fmt.Errorf("failed to create group: %w", err)
	}

	// 2. 批量添加成员 ( 异步 )。
	// 使用 Kafka 消息队列实现异步。
	if err := s.sendMemberAddMsg(ctx, group.ID, event.CreatorID, event.MemberIDs); err != nil {
		return domain.ChatGroup{}, fmt.Errorf("failed to send member add message: %w", err)
	}
	return group, nil
}

func (s *DefaultGroupService) AddMembers(ctx context.Context, event domain.GroupMemberAddEvent) error {
	return s.sendMemberAddMsg(ctx, event.GID, event.CreatorID, event.MemberIDs)
}

// sendMemberAddMsg 发送成员添加消息。
func (s *DefaultGroupService) sendMemberAddMsg(ctx context.Context, groupID, creatorID uint64, memberIDs []uint64) error {
	// 分批发送 ( 每批 500 个成员 )。
	const batchSize = 500
	memberCount := len(memberIDs)
	for i := 0; i < memberCount; i += batchSize {
		end := min(i+batchSize, memberCount)
		batch := memberIDs[i:end]

		msg := domain.GroupMemberAddEvent{
			GID:         groupID,
			CreatorID:   creatorID,
			MemberIDs:   batch,
			IsLastBatch: end >= memberCount, // 判断是否是最后一批成员。
		}

		val, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal channel member add event: %w", err)
		}

		if err := s.msgProducer.Produce(ctx, []byte(strconv.FormatUint(groupID, 10)), val); err != nil {
			// 发送失败，更新频道状态为 "failed"。
			_ = s.groupRepo.UpdateStatus(ctx, groupID, string(domain.ChatGroupStatusFailed))
			return fmt.Errorf("failed to produce channel member add event: %w", err)
		}
	}
	return nil
}

func (s *DefaultGroupService) ProcessAddMembersMsg(ctx context.Context, msg *xmq.Message) error {
	var event domain.GroupMemberAddEvent
	if err := json.Unmarshal(msg.Val, &event); err != nil {
		return fmt.Errorf("failed to unmarshal channel member add event: %w", err)
	}

	if err := s.addMembers(ctx, event.GID, event.CreatorID, event.MemberIDs); err != nil {
		// TODO: 需要增加成员添加失败处理。
		// 1. 指数退避重试 3 次 ( 1s -> 2s -> 4s )。
		// 2. 如果 3 次重试都失败，转发消息到死信队列 ( DLQ )。
	}

	if event.IsLastBatch {
		if err := s.groupRepo.UpdateStatus(ctx, event.GID, string(domain.ChatGroupStatusActive)); err != nil {
			s.logger.Error(
				"[chat-group-service] failed to update chat group status to active",
				zap.Uint64("group_id", event.GID),
				zap.Error(err),
			)
		}
		s.logger.Debug(
			"[chat-group-service] chat group activated successfully",
			zap.Uint64("group_id", event.GID),
		)
	}
	return nil
}

// addMembers 实际写入成员到数据库。
func (s *DefaultGroupService) addMembers(ctx context.Context, groupID, creatorID uint64, memberIDs []uint64) error {
	if err := s.groupMemberRepo.BatchCreate(ctx, groupID, creatorID, memberIDs); err != nil {
		return fmt.Errorf("failed to add members: %w", err)
	}
	return nil
}

package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
)

type GroupService interface {
	Create(ctx context.Context, event domain.GroupCreateEvent) (domain.ChatGroup, error)
	AddMembers(ctx context.Context, event domain.GroupMemberAddEvent) error

	ProcessAddMembersMsg(ctx context.Context, msg *xmq.Message) error
}

var _ GroupService = (*DefaultGroupService)(nil)

type DefaultGroupService struct {
	gmProducer mq.Producer

	groupRepo       repo.GroupRepo
	groupMemberRepo repo.GroupMemberRepo
}

func NewDefaultGroupService(
	gmProducer mq.Producer,
	groupRepo repo.GroupRepo,
	groupMemberRepo repo.GroupMemberRepo,
) (*DefaultGroupService, error) {
	return &DefaultGroupService{
		gmProducer: gmProducer,

		groupRepo:       groupRepo,
		groupMemberRepo: groupMemberRepo,
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
	// 群主必须并入成员批次——群主自身也是成员,
	// 否则其 user_group/channel 读侧行不会被建立, 群不会出现在自己的频道列表。
	memberIDs := make([]uint64, 0, len(event.MemberIDs)+1)
	memberIDs = append(memberIDs, event.CreatorID)
	for _, id := range event.MemberIDs {
		if id != event.CreatorID {
			memberIDs = append(memberIDs, id)
		}
	}

	if err := s.sendMemberAddMsg(ctx, group.ID, event.CreatorID, memberIDs); err != nil {
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

		if err := s.gmProducer.Produce(ctx, nil, val); err != nil {
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
		logging.For("chat-group-service").Error(
			"failed to add members, batch skipped",
			"group_id", event.GID,
			"error", err,
		)
	}

	if event.IsLastBatch {
		if err := s.groupRepo.UpdateStatus(ctx, event.GID, string(domain.ChatGroupStatusActive)); err != nil {
			logging.For("chat-group-service").Error(
				"failed to update chat group status to active",
				"group_id", event.GID,
				"error", err,
			)
		}
		logging.For("chat-group-service").Debug(
			"chat group activated successfully",
			"group_id", event.GID,
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

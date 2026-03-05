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

	// ProcessUserGroup 处理用户聊天群事件。
	// 用户于聊天群建立关联关系后，创建会话。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessUserGroup(ctx context.Context, msg *xmq.Message) error

	// ProcessGroupMember 处理聊天群成员添加事件。
	// 聊天群成员添加后，建立成员视图 ( 冗余成员信息 )。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessGroupMember(ctx context.Context, msg *xmq.Message) error

	// ProcessOutboxEvent 处理 outbox 事件。
	// 现有的 outbox 事件包含：
	// 	1. group_member 表的分流 ( 用于保证 user_group 的创建 )。
	// msg 是 debezium 推送到 kafka 的消息。
	ProcessOutboxEvent(ctx context.Context, msg *xmq.Message) error
}

var _ CDCService = (*DefaultCDCService)(nil)

type DefaultCDCService struct {
	rdb redis.Cmdable

	userRepo      repo.UserRepo
	userGroupRepo repo.UserGroupRepo

	groupRepo       repo.GroupRepo
	groupMemberRepo repo.GroupMemberRepo

	contactRepo repo.ContactRepo
	channelRepo repo.ChannelRepo

	logger *zap.Logger
}

func NewDefaultCDCService(
	rdb redis.Cmdable,
	userRepo repo.UserRepo,
	userGroupRepo repo.UserGroupRepo,
	groupRepo repo.GroupRepo,
	groupMemberRepo repo.GroupMemberRepo,
	contactRepo repo.ContactRepo,
	channelRepo repo.ChannelRepo,
	logger *zap.Logger,
) *DefaultCDCService {
	return &DefaultCDCService{
		rdb: rdb,

		userRepo:      userRepo,
		userGroupRepo: userGroupRepo,

		groupRepo:       groupRepo,
		groupMemberRepo: groupMemberRepo,

		contactRepo: contactRepo,
		channelRepo: channelRepo,

		logger: logger,
	}
}

func (s *DefaultCDCService) ProcessContactReview(ctx context.Context, msg *xmq.Message) error {
	var evt event.ContactReviewCDCEvent
	if err := json.Unmarshal(msg.Val, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal contact application cdc event: %w", err)
	}

	if !evt.IsApproved() {
		// 只处理审批通过事件。
		return nil
	}

	// 幂等性判断 ( 防止重复处理相同的联系人申请事件 )。
	cacheKey := fmt.Sprintf("contact:review:%d", evt.ID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug("[cdc-contact-review] contact review already processed", zap.Uint64("review_id", evt.ID))
		return nil
	}

	// 建立双向关联关系。
	eg, ctx := errgroup.WithContext(ctx)

	// 创建 A -> B 的联系人关系。
	eg.Go(func() error {
		return s.createContact(ctx, domain.Contact{
			UserID:     evt.ApplicantID,
			ContactID:  evt.TargetID,
			RemarkName: evt.ApplicantName,
			Source:     evt.Source,
			AddedAt:    evt.CreatedAt,
		})
	})

	// 创建 B -> A 的联系人关系。
	eg.Go(func() error {
		return s.createContact(ctx, domain.Contact{
			UserID:     evt.TargetID,
			ContactID:  evt.ApplicantID,
			RemarkName: evt.ApplicantName,
			Source:     evt.Source,
			AddedAt:    evt.ReviewedAt,
		})
	})

	if err := eg.Wait(); err != nil {
		s.logger.Error(
			"[cdc-contact-review] failed to create contact relation",
			zap.Uint64("review_id", evt.ID),
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
	exists, err := s.contactRepo.FindByUserIDAndContactID(ctx, contact.UserID, contact.ContactID)
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}

	if exists.ID != 0 && exists.DeletedAt != 0 {
		// 以前添加过联系人但被删除了，现在重新添加。
		if err := s.contactRepo.Recreate(ctx, exists); err != nil {
			return err
		}
		return nil
	}
	return s.contactRepo.Create(ctx, contact)
}

// createContactSync 创建联系人关系。
func (s *DefaultCDCService) ProcessContact(ctx context.Context, msg *xmq.Message) error {
	var evt event.ContactCDCEvent
	if err := json.Unmarshal(msg.Val, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal user contact cdc event: %w", err)
	}

	if !(evt.IsCreate() || evt.IsUpdate()) {
		// 只处理 create 和 update 事件。
		// - create: 创建联系人关系事件。
		// - update: 联系人被删除 ( 逻辑删除 DeletedAt != 0 )
		// 注意: 重新添加需要重新审批，所以不会触发 update 事件 ( DeletedAt != 0 -> DeletedAt = 0 )。
		return nil
	}

	if evt.IsUpdate() && evt.DeletedAt != 0 {
		// TODO: cdc 事件为删除用户联系人 ( 逻辑删除 )。
		return nil
	}

	// cdc 事件为创建联系人关系事件，创建会话。
	return s.createChannel(ctx, domain.Channel{
		UserID:      evt.UserID,
		TargetID:    evt.ContactID,
		ChannelType: domain.ChannelTypeSingle,
		OpenedAt:    evt.AddedAt,
	})
}

func (s *DefaultCDCService) ProcessUserGroup(ctx context.Context, msg *xmq.Message) error {
	var evt event.UserGroupCDCEvent
	if err := json.Unmarshal(msg.Val, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal user group cdc event: %w", err)
	}

	if !(evt.IsCreate() || evt.IsUpdate()) {
		// 只处理 create 和 update 事件。
		// - create
		//   创建联系人关系事件。
		// - update
		//   联系人被删除 ( 逻辑删除 LeftAt != 0 )。
		// 注意：
		//  重新添加需要重新审批，所以不会触发 update 事件 ( LeftAt != 0 -> LeftAt = 0 )。
		return nil
	}

	if evt.IsUpdate() && evt.LeftAt != 0 {
		// TODO: cdc 事件为聊天群成员离开事件 ( 逻辑删除 )。
		return nil
	}

	return s.createChannel(ctx, domain.Channel{
		UserID:      evt.UserID,
		TargetID:    evt.GroupID,
		ChannelType: domain.ChannelTypeGroup,
		OpenedAt:    evt.JoinedAt,
	})
}

func (s *DefaultCDCService) createChannel(ctx context.Context, channel domain.Channel) error {
	// 幂等性判断 ( 防止重复处理相同的联系人关系建立事件 )。
	cacheKey := fmt.Sprintf("channel:create:%s-%d-%d", channel.ChannelType, channel.UserID, channel.TargetID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-channel] channel already processed",
			zap.Uint64("user_id", channel.UserID),
			zap.Uint64("target_id", channel.TargetID),
		)
		return nil
	}

	// 填充会话冗余信息。
	if err := s.fillInChannelRedundant(ctx, &channel); err != nil {
		return err
	}

	// 创建会话 ( 这里消费的是联系人关系建立事件，只需要创建会话即可 )。
	if err := s.channelRepo.Create(ctx, channel); err != nil {
		s.logger.Error("[cdc-contact] failed to create channel for contact", zap.Error(err))
		return err
	}

	// 关联关系建立成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

// fillInChannelRedundant 填充会话冗余信息。
// 根据会话类型填充会话冗余信息。
// - 单聊：填充用户信息。
// - 群聊：填充群组信息。
func (s *DefaultCDCService) fillInChannelRedundant(ctx context.Context, channel *domain.Channel) error {
	switch channel.ChannelType {
	case domain.ChannelTypeSingle:
		user, err := s.userRepo.FindByID(ctx, channel.TargetID)
		if err != nil {
			if errors.Is(err, errs.ErrRecordNotFound) {
				s.logger.Error("[cdc-user-group] user not found", zap.Uint64("user_id", channel.TargetID), zap.Error(err))
				return err
			}
			return fmt.Errorf("failed to find user: %w", err)
		}
		channel.TargetInfoVer = user.InfoVer

		channel.ChannelName = user.Nickname
		channel.ChannelRemark = user.Nickname
		channel.ChannelAvatar = user.Avatar

	case domain.ChannelTypeGroup:
		group, err := s.groupRepo.FindByID(ctx, channel.TargetID)
		if err != nil {
			if errors.Is(err, errs.ErrRecordNotFound) {
				s.logger.Error("[cdc-user-group] chat group not found", zap.Uint64("group_id", channel.TargetID), zap.Error(err))
				return err
			}
			return fmt.Errorf("failed to find chat group: %w", err)
		}
		channel.TargetInfoVer = group.InfoVer

		channel.ChannelName = group.GroupName
		channel.ChannelRemark = group.GroupName
		channel.ChannelAvatar = group.GroupAvatar
	}
	return nil
}

func (s *DefaultCDCService) ProcessGroupMember(ctx context.Context, msg *xmq.Message) error {
	var evt event.GroupMemberCDCEvent
	if err := json.Unmarshal(msg.Val, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal group member cdc event: %w", err)
	}

	if !(evt.IsCreate() || evt.IsUpdate()) {
		// 只处理 create 和 update 事件。
		// - create
		// 	 聊天群成员添加事件。
		// - update
		// 	 1. 聊天群成员离开 ( 逻辑删除 LeftAt != 0 )。
		//   2. 聊天群成员重新加入 ( LeftAt != 0 -> LeftAt = 0 )。
		return nil
	}

	if evt.IsUpdate() && evt.LeftAt != 0 {
		// TODO: cdc 事件为聊天群成员离开事件 ( 逻辑删除 )。
		return nil
	}

	// cdc 事件为聊天群成员添加事件，创建聊天群成员视图。
	user, err := s.userRepo.FindByID(ctx, evt.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			s.logger.Error("[cdc-group-member] user not found", zap.Uint64("user_id", evt.UserID), zap.Error(err))
			return err
		}
		return fmt.Errorf("failed to find user: %w", err)
	}
	gmv := domain.GroupMemberView{
		ID: evt.ID,

		GroupID: evt.GroupID,
		UserID:  evt.UserID,

		MemberRole: domain.GroupMemberRole(evt.MemberRole),

		MemberNickname: user.Nickname,
		MemberAvatar:   user.Avatar,
		MemberGender:   user.Gender,
		MemberRemark:   user.Nickname,
		MemberInfoVer:  user.InfoVer,

		JoinedAt: evt.JoinedAt,
		LeftAt:   evt.LeftAt,
	}
	return s.createGmv(ctx, gmv)
}

func (s *DefaultCDCService) createGmv(ctx context.Context, gmv domain.GroupMemberView) error {
	// 幂等性判断 ( 防止重复处理相同的聊天群成员添加事件 )。
	cacheKey := fmt.Sprintf("group:member:create:%d-%d", gmv.GroupID, gmv.UserID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-group-member] group member already processed",
			zap.Uint64("group_id", gmv.GroupID),
			zap.Uint64("user_id", gmv.UserID),
		)
		return nil
	}

	if err := s.groupMemberRepo.CreateView(ctx, gmv); err != nil {
		return fmt.Errorf("failed to create group member view: %w", err)
	}

	// 聊天群成员视图创建成功，设置缓存标记 ( 用于幂等性判断 )。
	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

func (s *DefaultCDCService) ProcessOutboxEvent(ctx context.Context, msg *xmq.Message) error {
	var evt event.OutboxCDCEvent
	if err := json.Unmarshal(msg.Val, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal outbox cdc event: %w", err)
	}

	// 只处理创建事件。
	// 这里的创建事件是数据库表层面的。
	// 业务层面的操作来行由 EventType 来决定。
	if !evt.IsCreate() {
		return nil
	}

	var gmOutboxEvent domain.GroupMemberOutboxEvent
	if err := json.Unmarshal([]byte(evt.Payload), &gmOutboxEvent); err != nil {
		return fmt.Errorf("failed to unmarshal group member outbox event payload: %w", err)
	}

	cacheKey := fmt.Sprintf("group:member:outbox:%d-%d", gmOutboxEvent.GroupID, gmOutboxEvent.UserID)
	if exists, _ := s.rdb.Exists(ctx, cacheKey).Result(); exists > 0 {
		s.logger.Debug(
			"[cdc-group-member-outbox] group member outbox already processed",
			zap.Uint64("group_id", gmOutboxEvent.GroupID),
			zap.Uint64("user_id", gmOutboxEvent.UserID),
		)
		return nil
	}

	op := domain.OutboxEventType(evt.EventType)
	var err error
	switch op {
	case domain.OutboxEventTypeGmCreate:
		// 首次加入聊天群。
		err = s.userGroupRepo.Create(ctx, domain.UserGroup{
			UserID:   gmOutboxEvent.UserID,
			GroupID:  gmOutboxEvent.GroupID,
			JoinedAt: gmOutboxEvent.JoinedAt,
		})
	case domain.OutboxEventTypeGmUpdate:
		// 重新加入聊天群。
	case domain.OutboxEventTypeGmDelete:
		// 删除聊天群成员。
	default:
		return fmt.Errorf("unknown outbox event type: %s", op)
	}
	if err != nil {
		return fmt.Errorf("failed to process outbox event [ %s ]: %w", op, err)
	}

	s.rdb.Set(ctx, cacheKey, "1", cdcIdempotentCacheTTL)
	return nil
}

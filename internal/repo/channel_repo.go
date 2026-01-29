package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

type ChannelRepo interface {
	// Create 创建用户会话。
	Create(ctx context.Context, channel domain.Channel) error

	ListByUserID(ctx context.Context, userID uint64) ([]domain.Channel, error)
}

var _ ChannelRepo = (*DefaultChannelRepo)(nil)

type DefaultChannelRepo struct {
	userDao    dao.UserDao
	channelDao dao.ChannelDao
}

func NewDefaultChannelRepo(userDao dao.UserDao, channelDao dao.ChannelDao) *DefaultChannelRepo {
	return &DefaultChannelRepo{
		userDao:    userDao,
		channelDao: channelDao,
	}
}

func (r *DefaultChannelRepo) Create(ctx context.Context, channel domain.Channel) error {
	user, err := r.userDao.FindByID(ctx, channel.TargetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	channel.TargetInfoVer = user.InfoVer
	channel.ChannelName = user.Nickname
	channel.ChannelAvatar = user.Avatar

	err = r.channelDao.Create(ctx, r.toEntity(channel))
	if err != nil {
		return err
	}
	return nil
}

func (r *DefaultChannelRepo) ListByUserID(ctx context.Context, userID uint64) ([]domain.Channel, error) {
	entities, err := r.channelDao.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.Channel, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultChannelRepo) toEntity(channel domain.Channel) dao.Channel {
	return dao.Channel{
		ID: channel.ID,

		UserID:        channel.UserID,
		TargetID:      channel.TargetID,
		TargetInfoVer: channel.TargetInfoVer,

		ChannelType:   string(channel.ChannelType),
		ChannelName:   channel.ChannelName,
		ChannelRemark: channel.ChannelRemark,
		ChannelAvatar: channel.ChannelAvatar,

		IsMuted:   channel.IsMuted,
		IsPinned:  channel.IsPinned,
		IsHidden:  channel.IsHidden,
		IsStarred: channel.IsStarred,

		LastMessageID:         channel.LastMessageID,
		LastMessageType:       channel.LastMessageType,
		LastMessageContent:    channel.LastMessageContent,
		LastMessageSenderID:   channel.LastMessageSenderID,
		LastMessageSenderName: channel.LastMessageSenderName,
		LastMessageTime:       channel.LastMessageTime,

		UnreadCount:  channel.UnreadCount,
		MentionCount: channel.MentionCount,

		OpenedAt: channel.OpenedAt,
		ClosedAt: channel.ClosedAt,

		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}

func (r *DefaultChannelRepo) toDomain(channel dao.Channel) domain.Channel {
	return domain.Channel{
		ID: channel.ID,

		UserID:        channel.UserID,
		TargetID:      channel.TargetID,
		TargetInfoVer: channel.TargetInfoVer,

		ChannelType:   domain.ChannelType(channel.ChannelType),
		ChannelName:   channel.ChannelName,
		ChannelRemark: channel.ChannelRemark,
		ChannelAvatar: channel.ChannelAvatar,

		IsMuted:   channel.IsMuted,
		IsPinned:  channel.IsPinned,
		IsHidden:  channel.IsHidden,
		IsStarred: channel.IsStarred,

		LastMessageID:         channel.LastMessageID,
		LastMessageType:       channel.LastMessageType,
		LastMessageContent:    channel.LastMessageContent,
		LastMessageSenderID:   channel.LastMessageSenderID,
		LastMessageSenderName: channel.LastMessageSenderName,
		LastMessageTime:       channel.LastMessageTime,

		UnreadCount:  channel.UnreadCount,
		MentionCount: channel.MentionCount,

		OpenedAt: channel.OpenedAt,
		ClosedAt: channel.ClosedAt,

		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}

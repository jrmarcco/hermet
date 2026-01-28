package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type ChannelRepo interface {
	Create(ctx context.Context, channel domain.Channel) (domain.Channel, error)
	CreateWithMembers(ctx context.Context, channel domain.Channel, members []uint64) (domain.Channel, error)

	UpdateStatus(ctx context.Context, id uint64, status string) error
}

var _ ChannelRepo = (*DefaultChannelRepo)(nil)

type DefaultChannelRepo struct {
	channelDao dao.ChannelDao
}

func NewDefaultChannelRepo(channelDao dao.ChannelDao) *DefaultChannelRepo {
	return &DefaultChannelRepo{
		channelDao: channelDao,
	}
}

func (r *DefaultChannelRepo) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	entity, err := r.channelDao.Create(ctx, r.toEntity(channel))
	if err != nil {
		return domain.Channel{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultChannelRepo) CreateWithMembers(ctx context.Context, channel domain.Channel, members []uint64) (domain.Channel, error) {
	entity, err := r.channelDao.CreateWithMembers(ctx, r.toEntity(channel), members)
	if err != nil {
		return domain.Channel{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultChannelRepo) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.channelDao.UpdateStatus(ctx, id, status)
}

func (r *DefaultChannelRepo) toEntity(channel domain.Channel) dao.Channel {
	return dao.Channel{
		ID: channel.ID,

		ChannelName:        channel.ChannelName,
		ChannelAvatar:      channel.ChannelAvatar,
		ChannelStatus:      string(channel.ChannelStatus),
		ChannelMemberCount: channel.ChannelMemberCount,

		InfoVer: channel.InfoVer,

		LastMessageAt: channel.LastMessageAt,

		CreatorID: channel.CreatorID,

		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}

func (r *DefaultChannelRepo) toDomain(channel dao.Channel) domain.Channel {
	return domain.Channel{
		ID: channel.ID,

		ChannelName:        channel.ChannelName,
		ChannelAvatar:      channel.ChannelAvatar,
		ChannelStatus:      domain.ChannelStatus(channel.ChannelStatus),
		ChannelMemberCount: channel.ChannelMemberCount,

		InfoVer: channel.InfoVer,

		LastMessageAt: channel.LastMessageAt,

		CreatorID: channel.CreatorID,

		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}

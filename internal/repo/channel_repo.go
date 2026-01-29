package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type ChannelRepo interface {
	Create(ctx context.Context, channel domain.Channel) (domain.Channel, error)
	CreateMemberView(ctx context.Context, cmv domain.ChannelMemberView) (domain.ChannelMemberView, error)
	AddMembers(ctx context.Context, channelID uint64, memberIDs []uint64) error

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

func (r *DefaultChannelRepo) CreateMemberView(ctx context.Context, cmv domain.ChannelMemberView) (domain.ChannelMemberView, error) {
	entity, err := r.channelDao.CreateMemberView(ctx, r.toViewEntity(cmv))
	if err != nil {
		return domain.ChannelMemberView{}, err
	}
	return r.toViewDomain(entity), nil
}

func (r *DefaultChannelRepo) AddMembers(ctx context.Context, channelID uint64, memberIDs []uint64) error {
	return r.channelDao.AddMembers(ctx, channelID, memberIDs)
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

func (r *DefaultChannelRepo) toViewEntity(mv domain.ChannelMemberView) dao.ChannelMemberView {
	return dao.ChannelMemberView{
		ID: mv.ID,

		ChannelID: mv.CID,
		UserID:    mv.UID,

		MemberRole: string(mv.MemberRole),

		MemberNickname: mv.MemberNickname,
		MemberAvatar:   mv.MemberAvatar,
		MemberGender:   string(mv.MemberGender),
		MemberRemark:   mv.MemberRemark,
		MemberInfoVer:  mv.MemberInfoVer,

		JoinedAt: mv.JoinedAt,
		LeftAt:   mv.LeftAt,

		CreatedAt: mv.CreatedAt,
		UpdatedAt: mv.UpdatedAt,
	}
}

func (r *DefaultChannelRepo) toViewDomain(mv dao.ChannelMemberView) domain.ChannelMemberView {
	return domain.ChannelMemberView{
		ID: mv.ID,

		CID: mv.ChannelID,
		UID: mv.UserID,

		MemberRole: domain.ChannelMemberRole(mv.MemberRole),

		MemberNickname: mv.MemberNickname,
		MemberAvatar:   mv.MemberAvatar,
		MemberGender:   domain.UserGender(mv.MemberGender),
		MemberRemark:   mv.MemberRemark,
		MemberInfoVer:  mv.MemberInfoVer,

		JoinedAt: mv.JoinedAt,
		LeftAt:   mv.LeftAt,

		CreatedAt: mv.CreatedAt,
		UpdatedAt: mv.UpdatedAt,
	}
}

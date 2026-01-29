package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type GroupRepo interface {
	Create(ctx context.Context, group domain.ChatGroup) (domain.ChatGroup, error)
	CreateMemberView(ctx context.Context, mv domain.GroupMemberView) (domain.GroupMemberView, error)
	AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error

	UpdateStatus(ctx context.Context, id uint64, status string) error
}

var _ GroupRepo = (*DefaultGroupRepo)(nil)

type DefaultGroupRepo struct {
	groupDao dao.GroupDao
}

func NewDefaultGroupRepo(groupDao dao.GroupDao) *DefaultGroupRepo {
	return &DefaultGroupRepo{
		groupDao: groupDao,
	}
}

func (r *DefaultGroupRepo) Create(ctx context.Context, group domain.ChatGroup) (domain.ChatGroup, error) {
	entity, err := r.groupDao.Create(ctx, r.toEntity(group))
	if err != nil {
		return domain.ChatGroup{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultGroupRepo) CreateMemberView(ctx context.Context, mv domain.GroupMemberView) (domain.GroupMemberView, error) {
	entity, err := r.groupDao.CreateMemberView(ctx, r.toViewEntity(mv))
	if err != nil {
		return domain.GroupMemberView{}, err
	}
	return r.toViewDomain(entity), nil
}

func (r *DefaultGroupRepo) AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error {
	return r.groupDao.AddMembers(ctx, groupID, memberIDs)
}

func (r *DefaultGroupRepo) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.groupDao.UpdateStatus(ctx, id, status)
}

func (r *DefaultGroupRepo) toEntity(group domain.ChatGroup) dao.ChatGroup {
	return dao.ChatGroup{
		ID: group.ID,

		GroupName:        group.GroupName,
		GroupAvatar:      group.GroupAvatar,
		GroupStatus:      string(group.GroupStatus),
		GroupMemberCount: group.GroupMemberCount,

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

func (r *DefaultGroupRepo) toDomain(group dao.ChatGroup) domain.ChatGroup {
	return domain.ChatGroup{
		ID: group.ID,

		GroupName:        group.GroupName,
		GroupAvatar:      group.GroupAvatar,
		GroupStatus:      domain.ChatGroupStatus(group.GroupStatus),
		GroupMemberCount: group.GroupMemberCount,

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

func (r *DefaultGroupRepo) toViewEntity(mv domain.GroupMemberView) dao.GroupMemberView {
	return dao.GroupMemberView{
		ID: mv.ID,

		GroupID: mv.GID,
		UserID:  mv.UID,

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

func (r *DefaultGroupRepo) toViewDomain(mv dao.GroupMemberView) domain.GroupMemberView {
	return domain.GroupMemberView{
		ID: mv.ID,

		GID: mv.GroupID,
		UID: mv.UserID,

		MemberRole: domain.GroupMemberRole(mv.MemberRole),

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

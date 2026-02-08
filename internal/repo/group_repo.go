package repo

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type GroupRepo interface {
	Create(ctx context.Context, group domain.ChatGroup) (domain.ChatGroup, error)

	UpdateStatus(ctx context.Context, id uint64, status string) error

	FindByID(ctx context.Context, id uint64) (domain.ChatGroup, error)
}

var _ GroupRepo = (*DefaultGroupRepo)(nil)

type DefaultGroupRepo struct {
	userDao            dao.UserDao
	groupDao           dao.GroupDao
	groupMemberViewDao dao.GroupMemberViewDao
}

func NewDefaultGroupRepo(
	userDao dao.UserDao,
	groupDao dao.GroupDao,
	groupMemberViewDao dao.GroupMemberViewDao,
) *DefaultGroupRepo {
	return &DefaultGroupRepo{
		userDao:            userDao,
		groupDao:           groupDao,
		groupMemberViewDao: groupMemberViewDao,
	}
}

func (r *DefaultGroupRepo) Create(ctx context.Context, group domain.ChatGroup) (domain.ChatGroup, error) {
	entity, err := r.groupDao.Create(ctx, dao.ChatGroup{
		GroupName:   group.GroupName,
		GroupAvatar: group.GroupAvatar,
		GroupStatus: string(group.GroupStatus),

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	})
	if err != nil {
		return domain.ChatGroup{}, fmt.Errorf("failed to create group: %w", err)
	}
	return r.toDomain(entity), nil
}

func (r *DefaultGroupRepo) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.groupDao.UpdateStatus(ctx, id, status)
}

func (r *DefaultGroupRepo) FindByID(ctx context.Context, id uint64) (domain.ChatGroup, error) {
	group, err := r.groupDao.FindByID(ctx, id)
	if err != nil {
		return domain.ChatGroup{}, err
	}
	return r.toDomain(group), nil
}

func (r *DefaultGroupRepo) toDomain(group dao.ChatGroup) domain.ChatGroup {
	return domain.ChatGroup{
		ID: group.ID,

		GroupName:   group.GroupName,
		GroupAvatar: group.GroupAvatar,
		GroupStatus: domain.ChatGroupStatus(group.GroupStatus),

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		DeletedAt: group.DeletedAt,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

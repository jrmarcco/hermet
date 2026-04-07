package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type GroupApplicationRepo interface {
	Create(ctx context.Context, application domain.GroupApplication) error
}

var _ GroupApplicationRepo = (*DefaultGroupApplicationRepo)(nil)

type DefaultGroupApplicationRepo struct {
	groupApplicationDao dao.GroupApplicationDao
}

func NewDefaultGroupApplicationRepo(
	groupApplicationDao dao.GroupApplicationDao,
) *DefaultGroupApplicationRepo {
	return &DefaultGroupApplicationRepo{
		groupApplicationDao: groupApplicationDao,
	}
}

func (r *DefaultGroupApplicationRepo) Create(ctx context.Context, application domain.GroupApplication) error {
	_, err := r.groupApplicationDao.Create(ctx, r.toEntity(application))
	return err
}

func (r *DefaultGroupApplicationRepo) toEntity(application domain.GroupApplication) dao.GroupApplication {
	return dao.GroupApplication{
		ID: application.ID,

		ApplicantID: application.ApplicantID,

		GroupID:     application.GroupID,
		GroupName:   application.GroupName,
		GroupAvatar: application.GroupAvatar,

		ApplicationStatus:  string(application.ApplicationStatus),
		ApplicationMessage: application.ApplicationMessage,

		ReviewerID: application.ReviewerID,
		ReviewedAt: application.ReviewedAt,

		CreatedAt: application.CreatedAt,
		UpdatedAt: application.UpdatedAt,
	}
}

package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type GroupApplicationRepo interface {
	Create(ctx context.Context, application domain.GroupApplication) (domain.GroupApplication, error)
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

func (r *DefaultGroupApplicationRepo) Create(ctx context.Context, application domain.GroupApplication) (domain.GroupApplication, error) {
	entity, err := r.groupApplicationDao.Create(ctx, r.toEntity(application))
	if err != nil {
		return domain.GroupApplication{}, err
	}
	return r.toDomain(entity), nil
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

func (r *DefaultGroupApplicationRepo) toDomain(entity dao.GroupApplication) domain.GroupApplication {
	return domain.GroupApplication{
		ID: entity.ID,

		ApplicantID: entity.ApplicantID,

		GroupID:     entity.GroupID,
		GroupName:   entity.GroupName,
		GroupAvatar: entity.GroupAvatar,

		ApplicationStatus:  domain.ApplicationStatus(entity.ApplicationStatus),
		ApplicationMessage: entity.ApplicationMessage,

		ReviewerID: entity.ReviewerID,
		ReviewedAt: entity.ReviewedAt,

		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

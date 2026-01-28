package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type ContactApplicationRepo interface {
	Create(ctx context.Context, application domain.ContactApplication) (domain.ContactApplication, error)

	UpdateStatus(ctx context.Context, applicationID uint64, status domain.ApplicationStatus) error

	FindByID(ctx context.Context, applicationID uint64) (domain.ContactApplication, error)
	ListPendingByTargetID(ctx context.Context, targetID uint64) ([]domain.ContactApplication, error)
}

var _ ContactApplicationRepo = (*DefaultContactApplicationRepo)(nil)

type DefaultContactApplicationRepo struct {
	contactApplicationDao dao.ContactApplicationDao
}

func NewDefaultContactApplicationRepo(contactApplicationDao dao.ContactApplicationDao) *DefaultContactApplicationRepo {
	return &DefaultContactApplicationRepo{
		contactApplicationDao: contactApplicationDao,
	}
}

func (r *DefaultContactApplicationRepo) Create(ctx context.Context, application domain.ContactApplication) (domain.ContactApplication, error) {
	entity, err := r.contactApplicationDao.Create(ctx, r.toEntity(application))
	if err != nil {
		return domain.ContactApplication{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultContactApplicationRepo) UpdateStatus(ctx context.Context, applicationID uint64, status domain.ApplicationStatus) error {
	return r.contactApplicationDao.UpdateStatus(ctx, applicationID, status)
}

func (r *DefaultContactApplicationRepo) FindByID(ctx context.Context, applicationID uint64) (domain.ContactApplication, error) {
	entity, err := r.contactApplicationDao.FindByID(ctx, applicationID)
	if err != nil {
		return domain.ContactApplication{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultContactApplicationRepo) ListPendingByTargetID(ctx context.Context, targetID uint64) ([]domain.ContactApplication, error) {
	entities, err := r.contactApplicationDao.ListPendingByTargetID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.ContactApplication, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultContactApplicationRepo) toEntity(application domain.ContactApplication) dao.ContactApplication {
	return dao.ContactApplication{
		ID:                 application.ID,
		ApplicantID:        application.ApplicantID,
		TargetID:           application.TargetID,
		ApplicantName:      application.ApplicantName,
		ApplicantAvatar:    application.ApplicantAvatar,
		ApplicationStatus:  string(application.ApplicationStatus),
		ApplicationMessage: application.ApplicationMessage,
		Source:             string(application.Source),
		ReviewedAt:         application.ReviewedAt,
		CreatedAt:          application.CreatedAt,
		UpdatedAt:          application.UpdatedAt,
	}
}

func (r *DefaultContactApplicationRepo) toDomain(entity dao.ContactApplication) domain.ContactApplication {
	return domain.ContactApplication{
		ID:                 entity.ID,
		ApplicantID:        entity.ApplicantID,
		TargetID:           entity.TargetID,
		ApplicantName:      entity.ApplicantName,
		ApplicantAvatar:    entity.ApplicantAvatar,
		ApplicationStatus:  domain.ApplicationStatus(entity.ApplicationStatus),
		ApplicationMessage: entity.ApplicationMessage,
		Source:             domain.ContactSource(entity.Source),
		ReviewedAt:         entity.ReviewedAt,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}

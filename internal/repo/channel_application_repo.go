package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type ChannelApplicationRepo interface {
	Create(ctx context.Context, application domain.ChannelApplication) (domain.ChannelApplication, error)
}

var _ ChannelApplicationRepo = (*DefaultChannelApplicationRepo)(nil)

type DefaultChannelApplicationRepo struct {
	channelApplicationDao dao.ChannelApplicationDao
}

func NewDefaultChannelApplicationRepo(
	channelApplicationDao dao.ChannelApplicationDao,
) *DefaultChannelApplicationRepo {
	return &DefaultChannelApplicationRepo{
		channelApplicationDao: channelApplicationDao,
	}
}

func (r *DefaultChannelApplicationRepo) Create(ctx context.Context, application domain.ChannelApplication) (domain.ChannelApplication, error) {
	entity, err := r.channelApplicationDao.Create(ctx, r.toEntity(application))
	if err != nil {
		return domain.ChannelApplication{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultChannelApplicationRepo) toEntity(application domain.ChannelApplication) dao.ChannelApplication {
	return dao.ChannelApplication{
		ID: application.ID,

		ApplicantID: application.ApplicantID,

		ChannelID:     application.ChannelID,
		ChannelName:   application.ChannelName,
		ChannelAvatar: application.ChannelAvatar,

		ApplicationStatus:  string(application.ApplicationStatus),
		ApplicationMessage: application.ApplicationMessage,

		ReviewerID: application.ReviewerID,
		ReviewedAt: application.ReviewedAt,

		CreatedAt: application.CreatedAt,
		UpdatedAt: application.UpdatedAt,
	}
}

func (r *DefaultChannelApplicationRepo) toDomain(entity dao.ChannelApplication) domain.ChannelApplication {
	return domain.ChannelApplication{
		ID: entity.ID,

		ApplicantID: entity.ApplicantID,

		ChannelID:     entity.ChannelID,
		ChannelName:   entity.ChannelName,
		ChannelAvatar: entity.ChannelAvatar,

		ApplicationStatus:  domain.ApplicationStatus(entity.ApplicationStatus),
		ApplicationMessage: entity.ApplicationMessage,

		ReviewerID: entity.ReviewerID,
		ReviewedAt: entity.ReviewedAt,

		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

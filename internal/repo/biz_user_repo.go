package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

//go:generate mockgen -source=biz_user_repo.go -destination=mock/biz_user_repo.mock.go -package=repomock -typed BizUserRepo

type BizUserRepo interface {
	Create(ctx context.Context, user domain.BizUser) (domain.BizUser, error)

	FindByID(ctx context.Context, id uint64) (domain.BizUser, error)
	FindByEmail(ctx context.Context, email string) (domain.BizUser, error)
}

var _ BizUserRepo = (*DefaultBizUserRepo)(nil)

type DefaultBizUserRepo struct {
	dao dao.BizUserDao
}

func (r *DefaultBizUserRepo) Create(ctx context.Context, user domain.BizUser) (domain.BizUser, error) {
	entity, err := r.dao.Create(ctx, r.toEntity(user))
	if err != nil {
		return domain.BizUser{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultBizUserRepo) FindByID(ctx context.Context, id uint64) (domain.BizUser, error) {
	entity, err := r.dao.FindByID(ctx, id)
	if err != nil {
		// 将数据库层的错误转换为领域层错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BizUser{}, errs.ErrRecordNotFound
		}
		return domain.BizUser{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultBizUserRepo) FindByEmail(ctx context.Context, email string) (domain.BizUser, error) {
	entity, err := r.dao.FindByEmail(ctx, email)
	if err != nil {
		// 将数据库层的错误转换为领域层错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BizUser{}, errs.ErrRecordNotFound
		}
		return domain.BizUser{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultBizUserRepo) toEntity(user domain.BizUser) dao.BizUser {
	return dao.BizUser{
		ID: user.ID,

		Email:  user.Email,
		Mobile: user.Mobile,
		Passwd: user.Passwd,

		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Gender:   string(user.Gender),
		Region:   user.Region,
		Birthday: user.Birthday,
		Tagline:  user.Tagline,

		InfoVer: user.InfoVer,

		UserStatus: string(user.UserStatus),
		DeletedAt:  user.DeletedAt,

		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (r *DefaultBizUserRepo) toDomain(user dao.BizUser) domain.BizUser {
	return domain.BizUser{
		ID: user.ID,

		Email:  user.Email,
		Mobile: user.Mobile,
		Passwd: user.Passwd,

		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Gender:   domain.UserGender(user.Gender),
		Region:   user.Region,
		Birthday: user.Birthday,
		Tagline:  user.Tagline,

		InfoVer: user.InfoVer,

		UserStatus: domain.UserStatus(user.UserStatus),
		DeletedAt:  user.DeletedAt,

		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func NewDefaultBizUserRepo(dao dao.BizUserDao) *DefaultBizUserRepo {
	return &DefaultBizUserRepo{dao: dao}
}

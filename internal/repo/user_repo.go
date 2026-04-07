package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

//go:generate mockgen -source=user_repo.go -destination=mock/user_repo.mock.go -package=repomock -typed UserRepo

type UserRepo interface {
	Create(ctx context.Context, user domain.BizUser) error

	FindByID(ctx context.Context, id uint64) (domain.BizUser, error)
	FindByEmail(ctx context.Context, email string) (domain.BizUser, error)
}

var _ UserRepo = (*DefaultUserRepo)(nil)

type DefaultUserRepo struct {
	userDao dao.UserDao
}

func NewDefaultUserRepo(userDao dao.UserDao) *DefaultUserRepo {
	return &DefaultUserRepo{userDao: userDao}
}

func (r *DefaultUserRepo) Create(ctx context.Context, user domain.BizUser) error {
	_, err := r.userDao.Create(ctx, r.toEntity(user))
	return err
}

func (r *DefaultUserRepo) FindByID(ctx context.Context, id uint64) (domain.BizUser, error) {
	entity, err := r.userDao.FindByID(ctx, id)
	if err != nil {
		// 将数据库层的错误转换为领域层错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BizUser{}, errs.ErrRecordNotFound
		}
		return domain.BizUser{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultUserRepo) FindByEmail(ctx context.Context, email string) (domain.BizUser, error) {
	entity, err := r.userDao.FindByEmail(ctx, email)
	if err != nil {
		// 将数据库层的错误转换为领域层错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BizUser{}, errs.ErrRecordNotFound
		}
		return domain.BizUser{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultUserRepo) toEntity(user domain.BizUser) dao.BizUser {
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

func (r *DefaultUserRepo) toDomain(user dao.BizUser) domain.BizUser {
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

package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

type UserContactRepo interface {
	// Create 创建用户联系人。
	Create(ctx context.Context, userContact domain.UserContact) (domain.UserContact, error)

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.UserContact, error)
	ListViewByUserID(ctx context.Context, userID uint64) ([]domain.UserContactView, error)
}

var _ UserContactRepo = (*DefaultUserContactRepo)(nil)

type DefaultUserContactRepo struct {
	bizUserDao     dao.BizUserDao
	userContactDao dao.UserContactDao
}

func NewDefaultUserContactRepo(
	bizUserDao dao.BizUserDao,
	userContactDao dao.UserContactDao,
) *DefaultUserContactRepo {
	return &DefaultUserContactRepo{
		bizUserDao:     bizUserDao,
		userContactDao: userContactDao,
	}
}

func (r *DefaultUserContactRepo) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.UserContact, error) {
	entity, err := r.userContactDao.FindByUserIDAndContactID(ctx, userID, contactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserContact{}, errs.ErrRecordNotFound
		}
		return domain.UserContact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultUserContactRepo) Create(ctx context.Context, userContact domain.UserContact) (domain.UserContact, error) {
	// 查询联系人信息 ( 用于冗余联系人信息 )。
	user, err := r.bizUserDao.FindByID(ctx, userContact.ContactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserContact{}, errs.ErrRecordNotFound
		}
		return domain.UserContact{}, err
	}

	entity, err := r.userContactDao.Create(ctx, r.toEntity(userContact), user)
	if err != nil {
		return domain.UserContact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultUserContactRepo) ListViewByUserID(ctx context.Context, userID uint64) ([]domain.UserContactView, error) {
	entities, err := r.userContactDao.ListViewByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.UserContactView, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toViewDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultUserContactRepo) toEntity(uc domain.UserContact) dao.UserContact {
	return dao.UserContact{
		ID: uc.ID,

		UserID:    uc.UserID,
		ContactID: uc.ContactID,

		RemarkName: uc.RemarkName,
		Source:     string(uc.Source),

		Tags: uc.Tags,

		IsStarred: uc.IsStarred,
		IsBlocked: uc.IsBlocked,

		AddedAt:   uc.AddedAt,
		DeletedAt: uc.DeletedAt,

		CreatedAt: uc.CreatedAt,
		UpdatedAt: uc.UpdatedAt,
	}
}

func (r *DefaultUserContactRepo) toDomain(uc dao.UserContact) domain.UserContact {
	return domain.UserContact{
		ID: uc.ID,

		UserID:    uc.UserID,
		ContactID: uc.ContactID,

		RemarkName: uc.RemarkName,
		Source:     domain.UserContactSource(uc.Source),

		Tags: uc.Tags,

		IsStarred: uc.IsStarred,
		IsBlocked: uc.IsBlocked,

		AddedAt:   uc.AddedAt,
		DeletedAt: uc.DeletedAt,

		CreatedAt: uc.CreatedAt,
		UpdatedAt: uc.UpdatedAt,
	}
}

func (r *DefaultUserContactRepo) toViewDomain(ucv dao.UserContactView) domain.UserContactView {
	return domain.UserContactView{
		ID: ucv.ID,

		UserID:    ucv.UserID,
		ContactID: ucv.ContactID,

		ContactNickname: ucv.ContactNickname,
		ContactAvatar:   ucv.ContactAvatar,
		ContactInfoVer:  ucv.ContactInfoVer,

		RemarkName: ucv.RemarkName,
		Tags:       ucv.Tags,

		IsStarred: ucv.IsStarred,
		IsBlocked: ucv.IsBlocked,

		Mobile: ucv.Mobile,
		Source: domain.UserContactSource(ucv.Source),

		AddedAt: ucv.AddedAt,
	}
}

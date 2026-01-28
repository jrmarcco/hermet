package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

type ContactRepo interface {
	// Create 创建用户联系人。
	Create(ctx context.Context, contact domain.Contact) (domain.Contact, error)
	// Recreate 重新创建用户联系人 ( 用于删除后重新添加 )。
	Recreate(ctx context.Context, contact domain.Contact) (domain.Contact, error)

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.Contact, error)
	ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ContactView, error)
}

var _ ContactRepo = (*DefaultContactRepo)(nil)

type DefaultContactRepo struct {
	bizUserDao     dao.BizUserDao
	userContactDao dao.ContactDao
}

func NewDefaultContactRepo(
	bizUserDao dao.BizUserDao,
	userContactDao dao.ContactDao,
) *DefaultContactRepo {
	return &DefaultContactRepo{
		bizUserDao:     bizUserDao,
		userContactDao: userContactDao,
	}
}

func (r *DefaultContactRepo) Create(ctx context.Context, contact domain.Contact) (domain.Contact, error) {
	// 查询联系人信息 ( 用于冗余联系人信息 )。
	user, err := r.bizUserDao.FindByID(ctx, contact.ContactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Contact{}, errs.ErrRecordNotFound
		}
		return domain.Contact{}, err
	}

	entity, err := r.userContactDao.Create(ctx, r.toEntity(contact), user)
	if err != nil {
		return domain.Contact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultContactRepo) Recreate(ctx context.Context, contact domain.Contact) (domain.Contact, error) {
	entity, err := r.userContactDao.Recreate(ctx, r.toEntity(contact))
	if err != nil {
		return domain.Contact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultContactRepo) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.Contact, error) {
	entity, err := r.userContactDao.FindByUserIDAndContactID(ctx, userID, contactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Contact{}, errs.ErrRecordNotFound
		}
		return domain.Contact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *DefaultContactRepo) ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ContactView, error) {
	entities, err := r.userContactDao.ListViewByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.ContactView, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toViewDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultContactRepo) toEntity(uc domain.Contact) dao.Contact {
	return dao.Contact{
		ID: uc.ID,

		UserID:    uc.UserID,
		ContactID: uc.ContactID,

		RemarkName: uc.RemarkName,
		Source:     string(uc.Source),

		Tags: uc.Tags,

		IsMuted:  uc.IsMuted,
		IsPinned: uc.IsPinned,

		AddedAt:   uc.AddedAt,
		DeletedAt: uc.DeletedAt,

		CreatedAt: uc.CreatedAt,
		UpdatedAt: uc.UpdatedAt,
	}
}

func (r *DefaultContactRepo) toDomain(uc dao.Contact) domain.Contact {
	return domain.Contact{
		ID: uc.ID,

		UserID:    uc.UserID,
		ContactID: uc.ContactID,

		RemarkName: uc.RemarkName,
		Source:     domain.ContactSource(uc.Source),

		Tags: uc.Tags,

		IsMuted:  uc.IsMuted,
		IsPinned: uc.IsPinned,

		AddedAt:   uc.AddedAt,
		DeletedAt: uc.DeletedAt,

		CreatedAt: uc.CreatedAt,
		UpdatedAt: uc.UpdatedAt,
	}
}

func (r *DefaultContactRepo) toViewDomain(ucv dao.ContactView) domain.ContactView {
	return domain.ContactView{
		ID: ucv.ID,

		UserID:    ucv.UserID,
		ContactID: ucv.ContactID,

		ContactNickname: ucv.ContactNickname,
		ContactAvatar:   ucv.ContactAvatar,
		ContactInfoVer:  ucv.ContactInfoVer,

		RemarkName: ucv.RemarkName,
		Tags:       ucv.Tags,

		IsMuted:  ucv.IsMuted,
		IsPinned: ucv.IsPinned,

		Mobile: ucv.Mobile,
		Source: domain.ContactSource(ucv.Source),

		AddedAt: ucv.AddedAt,
	}
}

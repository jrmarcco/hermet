package dao

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Contact 【 写入侧 】用户联系人表。
// 以 UserID 为分片键。
type Contact struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	RemarkName string `gorm:"column:remark_name"`
	Source     string `gorm:"column:source"`

	IsStarred bool `gorm:"column:is_starred"`
	IsBlocked bool `gorm:"column:is_blocked"`
	IsMuted   bool `gorm:"column:is_muted"`
	IsPinned  bool `gorm:"column:is_pinned"`

	AddedAt   int64 `gorm:"column:added_at"`
	DeletedAt int64 `gorm:"column:deleted_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ContactDao interface {
	// CreateInTx 创建用户联系人。
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, contact Contact) error
	// RecreateInTx 重新创建用户联系人 ( 用于删除后重新添加 )。
	RecreateInTx(ctx context.Context, tx *gorm.DB, tbName string, contact Contact) error

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (Contact, error)
}

var _ ContactDao = (*DefaultContactDao)(nil)

type DefaultContactDao struct {
	dbs *xsync.Map[string, *gorm.DB]

	shardHelper *sharding.ShardHelper
}

func NewDefaultContactDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultContactDao {
	return &DefaultContactDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultContactDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, contact Contact) error {
	return tx.WithContext(ctx).Table(tbName).Model(&Contact{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: "contact_id"}},
			DoNothing: true,
		}).
		Create(&contact).Error
}

func (d *DefaultContactDao) RecreateInTx(ctx context.Context, tx *gorm.DB, tbName string, contact Contact) error {
	return tx.WithContext(ctx).Table(tbName).Model(&Contact{}).
		Where("id = ?", contact.ID).
		UpdateColumns(map[string]any{
			"deleted_at": 0,
			"added_at":   contact.AddedAt,
			colUpdatedAt: contact.UpdatedAt,
		}).Error
}

func (d *DefaultContactDao) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (Contact, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return Contact{}, err
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return Contact{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var uc Contact
	err = db.WithContext(ctx).Table(dst.TB).Model(&Contact{}).
		Where("user_id = ?", userID).
		Where("contact_id = ?", contactID).
		First(&uc).Error
	if err != nil {
		return Contact{}, err
	}
	return uc, nil
}

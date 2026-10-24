package dao

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ContactView 【 读取侧 】用户联系人视图表。
// 以 UserID 为分片键。
type ContactView struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	ContactNickname string `gorm:"column:contact_nickname"`
	ContactAvatar   string `gorm:"column:contact_avatar"`
	ContactGender   string `gorm:"column:contact_gender"`
	ContactMobile   string `gorm:"column:contact_mobile"`
	ContactInfoVer  int    `gorm:"column:contact_info_ver"`

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

type ContactViewDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cv ContactView) error
	RecreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cv ContactView) error

	ListByUserID(ctx context.Context, userID uint64) ([]ContactView, error)
}

var _ ContactViewDao = (*DefaultContactViewDao)(nil)

type DefaultContactViewDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultContactViewDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultContactViewDao {
	return &DefaultContactViewDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultContactViewDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cv ContactView) error {
	return tx.WithContext(ctx).Table(tbName).Model(&ContactView{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: "contact_id"}},
			DoNothing: true,
		}).
		Create(&cv).Error
}

func (d *DefaultContactViewDao) RecreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cv ContactView) error {
	return tx.WithContext(ctx).Table(tbName).Model(&ContactView{}).
		Where("id = ?", cv.ID).
		UpdateColumns(map[string]any{
			// 更新联系人信息。
			"contact_nickname": cv.ContactNickname,
			"contact_avatar":   cv.ContactAvatar,
			"contact_gender":   cv.ContactGender,
			"contact_mobile":   cv.ContactMobile,
			"contact_info_ver": cv.ContactInfoVer,
			"source":           cv.Source,

			"is_starred": cv.IsStarred,
			"is_blocked": cv.IsBlocked,
			"is_muted":   cv.IsMuted,
			"is_pinned":  cv.IsPinned,

			// 重置逻辑删除字段。
			"deleted_at": 0,
			// 更新添加时间以及更新时间。
			"added_at":   cv.AddedAt,
			colUpdatedAt: cv.UpdatedAt,
		}).Error
}

func (d *DefaultContactViewDao) ListByUserID(ctx context.Context, userID uint64) ([]ContactView, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var views []ContactView
	err = db.WithContext(ctx).Table(dst.TB).Model(&ContactView{}).
		Where("user_id = ?", userID).
		Where("deleted_at = ?", 0).
		Find(&views).Error
	return views, err
}

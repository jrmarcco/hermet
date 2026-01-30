package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserGroup struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	GroupID uint64 `gorm:"column:group_id"`

	RemarkName string `gorm:"column:remark_name"`

	IsMuted  bool `gorm:"column:is_muted"`
	IsPinned bool `gorm:"column:is_pinned"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type UserGroupDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ug UserGroup) error
}

var _ UserGroupDao = (*DefaultUserGroupDao)(nil)

type DefaultUserGroupDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserGroupDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultUserGroupDao {
	return &DefaultUserGroupDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultUserGroupDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ug UserGroup) error {
	return tx.WithContext(ctx).Table(tbName).Model(&UserGroup{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "group_id"}},
			DoNothing: true,
		}).
		Create(&ug).Error
}

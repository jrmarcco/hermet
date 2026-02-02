package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserGroupView struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	GroupID uint64 `gorm:"column:group_id"`

	GroupName    string `gorm:"column:group_name"`
	GroupAvatar  string `gorm:"column:group_avatar"`
	GroupStatus  string `gorm:"column:group_status"`
	GroupInfoVer int    `gorm:"column:group_info_ver"`

	RemarkName string `gorm:"column:remark_name"`

	IsMuted  bool `gorm:"column:is_muted"`
	IsPinned bool `gorm:"column:is_pinned"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type UserGroupViewDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ugv UserGroupView) error
}

var _ UserGroupViewDao = (*DefaultUserGroupViewDao)(nil)

type DefaultUserGroupViewDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserGroupViewDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultUserGroupViewDao {
	return &DefaultUserGroupViewDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultUserGroupViewDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ugv UserGroupView) error {
	return tx.WithContext(ctx).Table(tbName).Model(&UserGroupView{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "group_id"}},
			DoNothing: true,
		}).
		Create(&ugv).Error
}

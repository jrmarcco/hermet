package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupMemberView struct {
	ID uint64 `gorm:"column:id"`

	GroupID uint64 `gorm:"column:group_id"`
	UserID  uint64 `gorm:"column:user_id"`

	MemberRole string `gorm:"column:member_role"`

	MemberNickname string `gorm:"column:member_nickname"`
	MemberAvatar   string `gorm:"column:member_avatar"`
	MemberGender   string `gorm:"column:member_gender"`
	MemberRemark   string `gorm:"column:member_remark"`
	MemberInfoVer  int    `gorm:"column:member_info_ver"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type GroupMemberViewDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, gmv GroupMemberView) error
}

var _ GroupMemberViewDao = (*DefaultGroupMemberViewDao)(nil)

type DefaultGroupMemberViewDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultGroupMemberViewDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultGroupMemberViewDao {
	return &DefaultGroupMemberViewDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultGroupMemberViewDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, gmv GroupMemberView) error {
	return tx.WithContext(ctx).Table(tbName).Model(&GroupMemberView{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).
		Create(&gmv).Error
}

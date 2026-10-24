package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserGroupReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	GroupID uint64 `gorm:"column:group_id"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type UserGroupRidxDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ugri UserGroupReverseIndex) error
}

type DefaultUserGroupRidxDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserGroupRidxDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultUserGroupRidxDao {
	return &DefaultUserGroupRidxDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultUserGroupRidxDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ugri UserGroupReverseIndex) error {
	return tx.WithContext(ctx).Table(tbName).Model(&UserGroupReverseIndex{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: colGroupID}},
			DoNothing: true,
		}).
		Create(&ugri).Error
}

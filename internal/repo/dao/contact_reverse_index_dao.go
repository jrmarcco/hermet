package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ContactReverseIndex 联系人反向索引表。
type ContactReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	OwnerID uint64 `gorm:"column:owner_id"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type ContactRidxDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cri ContactReverseIndex) error
}

var _ ContactRidxDao = (*DefaultContactRidxDao)(nil)

type DefaultContactRidxDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultContactRidxDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultContactRidxDao {
	return &DefaultContactRidxDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultContactRidxDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, cri ContactReverseIndex) error {
	return tx.WithContext(ctx).Table(tbName).Model(&ContactReverseIndex{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: "owner_id"}},
			DoNothing: true,
		}).
		Create(&cri).Error
}

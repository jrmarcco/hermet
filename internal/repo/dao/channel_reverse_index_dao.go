package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChannelReverseIndex 频道反向索引表。
type ChannelReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID   uint64 `gorm:"column:user_id"`
	TargetID uint64 `gorm:"column:target_id"`

	ChannelType string `gorm:"column:channel_type"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type ChannelRidxDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, channelRidx ChannelReverseIndex) error
}

var _ ChannelRidxDao = (*DefaultChannelRidxDao)(nil)

type DefaultChannelRidxDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultChannelRidxDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultChannelRidxDao {
	return &DefaultChannelRidxDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultChannelRidxDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, channelRidx ChannelReverseIndex) error {
	return tx.WithContext(ctx).Table(tbName).Model(&ChannelReverseIndex{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: "target_id"}, {Name: "channel_type"}},
			DoNothing: true,
		}).
		Create(&channelRidx).Error
}

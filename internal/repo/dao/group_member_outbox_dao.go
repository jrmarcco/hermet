package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupMemberOutboxEvent struct {
	ID uint64 `gorm:"column:id"`

	GroupID uint64 `gorm:"column:group_id"`
	UserID  uint64 `gorm:"column:user_id"`

	OpType string `gorm:"column:op_type"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type GroupMemberOutboxDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, gmo GroupMemberOutboxEvent) error
}

var _ GroupMemberOutboxDao = (*DefaultGroupMemberOutboxDao)(nil)

type DefaultGroupMemberOutboxDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultGroupMemberOutboxDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultGroupMemberOutboxDao {
	return &DefaultGroupMemberOutboxDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultGroupMemberOutboxDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, gmo GroupMemberOutboxEvent) error {
	return tx.WithContext(ctx).Table(tbName).Model(&GroupMemberOutboxEvent{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}, {Name: "op_type"}},
			DoNothing: true,
		}).
		Create(&gmo).Error
}

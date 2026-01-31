package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxEvent struct {
	ID uint64 `gorm:"column:id"`

	AggregateID   uint64 `gorm:"column:aggregate_id"`
	AggregateType string `gorm:"column:aggregate_type"`

	EventType string `gorm:"column:event_type"`
	Payload   []byte `gorm:"column:payload"`
	Metadata  []byte `gorm:"column:metadata"`

	RetryCount int `gorm:"column:retry_count"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type OutboxEventDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, outboxEvent OutboxEvent) error
}

var _ OutboxEventDao = (*DefaultOutboxEventDao)(nil)

type DefaultOutboxEventDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultOutboxEventDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultOutboxEventDao {
	return &DefaultOutboxEventDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultOutboxEventDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, outboxEvent OutboxEvent) error {
	return tx.WithContext(ctx).Table(tbName).Model(&OutboxEvent{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "aggregate_id"}, {Name: "aggregate_type"}, {Name: "event_type"}},
			DoNothing: true,
		}).
		Create(&outboxEvent).Error
}

package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// ConversationSeq 会话 seq 分配表。
type ConversationSeq struct {
	BusinessID     uint64 `gorm:"column:business_id"`
	ConversationID string `gorm:"column:conversation_id"`

	Seq int64 `gorm:"column:seq"`

	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ConversationSeqDao interface {
	// NextSeqInTx 事务内原子分配 n 个连续 conversation_seq，返回区间 [start, start+n-1]。
	// 与消息正本写在同一分片事务内完成 ( 同布局路由 )，保证 seq 与正本的原子性。
	NextSeqInTx(ctx context.Context, tx *gorm.DB, tbName string, businessID uint64, conversationID string, n int64) (start int64, err error)

	// LatestSeq 返回会话 timeline 当前最大 conversation_seq ( 未分配过时为 0 )。
	LatestSeq(ctx context.Context, businessID uint64, conversationID string) (int64, error)
}

var _ ConversationSeqDao = (*DefaultConversationSeqDao)(nil)

type DefaultConversationSeqDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultConversationSeqDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultConversationSeqDao {
	return &DefaultConversationSeqDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultConversationSeqDao) NextSeqInTx(
	ctx context.Context,
	tx *gorm.DB,
	tbName string,
	businessID uint64,
	conversationID string,
	n int64,
) (int64, error) {
	now := time.Now().UnixMilli()

	// upsert + 行锁自增，RETURNING 返回自增后的最大 seq。
	// 表名由分片框架生成 ( 前缀_数字 格式 )，非用户输入。
	var maxSeq int64
	err := tx.WithContext(ctx).Raw(
		fmt.Sprintf(
			`INSERT INTO %s (business_id, conversation_id, seq, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT (business_id, conversation_id)
			 DO UPDATE SET seq = %s.seq + ?, updated_at = ?
			 RETURNING seq`,
			tbName, tbName,
		),
		businessID, conversationID, n, now, n, now,
	).Scan(&maxSeq).Error
	if err != nil {
		return 0, fmt.Errorf("failed to allocate conversation seq: %w", err)
	}

	return maxSeq - n + 1, nil
}

func (d *DefaultConversationSeqDao) LatestSeq(ctx context.Context, businessID uint64, conversationID string) (int64, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewStringSharder(conversationID))
	if err != nil {
		return 0, fmt.Errorf("failed to get shard destination from conversation id [ %s ]: %w", conversationID, err)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return 0, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var seq int64
	err = db.WithContext(ctx).Table(dst.TB).Model(&ConversationSeq{}).
		Select("COALESCE(seq, 0)").
		Where("business_id = ?", businessID).
		Where("conversation_id = ?", conversationID).
		Scan(&seq).Error
	return seq, err
}

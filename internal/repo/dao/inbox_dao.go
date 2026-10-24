package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InboxEntryKind 收件箱条目类型。
type InboxEntryKind int16

const (
	// InboxEntryKindChat 写扩散条目 ( 消息本体内联 )。
	InboxEntryKindChat InboxEntryKind = 1
	// InboxEntryKindPointer 读扩散指针 ( 会话更新 )。
	InboxEntryKindPointer InboxEntryKind = 2
)

// InboxSeq 收件箱 seq 分配表。
type InboxSeq struct {
	BusinessID uint64 `gorm:"column:business_id"`
	UserID     uint64 `gorm:"column:user_id"`

	Seq int64 `gorm:"column:seq"`

	UpdatedAt int64 `gorm:"column:updated_at"`
}

// InboxEntry 收件箱条目表。
type InboxEntry struct {
	BusinessID uint64 `gorm:"column:business_id"`
	UserID     uint64 `gorm:"column:user_id"`
	Seq        int64  `gorm:"column:seq"`

	Kind InboxEntryKind `gorm:"column:kind"`

	MessageID string `gorm:"column:message_id"`

	ConversationID        string `gorm:"column:conversation_id"`
	ConversationLatestSeq int64  `gorm:"column:conversation_latest_seq"`

	SenderID    uint64 `gorm:"column:sender_id"`
	Body        []byte `gorm:"column:body"`
	ContentType string `gorm:"column:content_type"`
	SentAt      int64  `gorm:"column:sent_at"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type InboxDao interface {
	// NextSeqInTx 事务内原子分配 n 个连续 seq，返回区间 [start, start+n-1]。
	// 同一分区内同一发送者的消息由 Kafka 分区序保证串行消费，
	// 跨发送者并发分配由行锁保证原子性。
	NextSeqInTx(ctx context.Context, tx *gorm.DB, tbName string, businessID, userID uint64, n int64) (start int64, err error)

	// CreateEntryInTx 事务内写入收件箱条目 ( 按 ( business_id, user_id, message_id ) 幂等 )。
	CreateEntryInTx(ctx context.Context, tx *gorm.DB, tbName string, entry InboxEntry) error

	// ListEntries 返回 ( cursor, +inf ] 内按 seq 升序的至多 limit 条条目。
	ListEntries(ctx context.Context, businessID, userID uint64, cursor int64, limit int) ([]InboxEntry, error)

	// LatestSeq 返回该用户收件箱当前最大 seq ( 未分配过时为 0 )。
	LatestSeq(ctx context.Context, businessID, userID uint64) (int64, error)
}

var _ InboxDao = (*DefaultInboxDao)(nil)

type DefaultInboxDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultInboxDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultInboxDao {
	return &DefaultInboxDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultInboxDao) NextSeqInTx(
	ctx context.Context,
	tx *gorm.DB,
	tbName string,
	businessID, userID uint64,
	n int64,
) (int64, error) {
	now := time.Now().UnixMilli()

	// upsert + 行锁自增，RETURNING 返回自增后的最大 seq。
	// 表名由分片框架生成 ( 前缀_数字 格式 )，非用户输入。
	var maxSeq int64
	err := tx.WithContext(ctx).Raw(
		fmt.Sprintf(
			`INSERT INTO %s (business_id, user_id, seq, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT (business_id, user_id)
			 DO UPDATE SET seq = %s.seq + ?, updated_at = ?
			 RETURNING seq`,
			tbName, tbName,
		),
		businessID, userID, n, now, n, now,
	).Scan(&maxSeq).Error
	if err != nil {
		return 0, fmt.Errorf("failed to allocate inbox seq: %w", err)
	}

	return maxSeq - n + 1, nil
}

func (d *DefaultInboxDao) CreateEntryInTx(
	ctx context.Context,
	tx *gorm.DB,
	tbName string,
	entry InboxEntry,
) error {
	return tx.WithContext(ctx).Table(tbName).Model(&InboxEntry{}).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "business_id"}, {Name: colUserID}, {Name: "message_id"},
			},
			DoNothing: true,
		}).
		Create(&entry).Error
}

func (d *DefaultInboxDao) ListEntries(
	ctx context.Context,
	businessID, userID uint64,
	cursor int64,
	limit int,
) ([]InboxEntry, error) {
	dst, db, err := d.dst(userID)
	if err != nil {
		return nil, err
	}

	entries := make([]InboxEntry, 0, limit)
	err = db.WithContext(ctx).Table(dst.TB).Model(&InboxEntry{}).
		Where("business_id = ?", businessID).
		Where("user_id = ?", userID).
		Where("seq > ?", cursor).
		Order("seq ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

func (d *DefaultInboxDao) LatestSeq(ctx context.Context, businessID, userID uint64) (int64, error) {
	dst, db, err := d.dst(userID)
	if err != nil {
		return 0, err
	}

	var seq int64
	err = db.WithContext(ctx).Table(dst.TB).Model(&InboxSeq{}).
		Select("COALESCE(seq, 0)").
		Where("business_id = ?", businessID).
		Where("user_id = ?", userID).
		Scan(&seq).Error
	return seq, err
}

func (d *DefaultInboxDao) dst(userID uint64) (sharding.Dst, *gorm.DB, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return sharding.Dst{}, nil, fmt.Errorf("failed to get shard destination from user id [ %d ]: %w", userID, err)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return sharding.Dst{}, nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}
	return dst, db, nil
}

package dao

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Message 消息正本表。
type Message struct {
	ID uint64 `gorm:"column:id"`

	MessageID  string `gorm:"column:message_id"`
	BusinessID uint64 `gorm:"column:business_id"`

	ConversationID  string `gorm:"column:conversation_id"`
	ConversationSeq int64  `gorm:"column:conversation_seq"`
	SenderID        uint64 `gorm:"column:sender_id"`

	Body        []byte `gorm:"column:body"`
	ContentType string `gorm:"column:content_type"`

	SentAt    int64 `gorm:"column:sent_at"`
	CreatedAt int64 `gorm:"column:created_at"`
}

type MessageDao interface {
	// SaveInTx 事务内写入消息正本 ( 按 ( business_id, message_id ) 幂等 )。
	SaveInTx(ctx context.Context, tx *gorm.DB, tbName string, message Message) error

	// ListByConversation 返回会话 timeline 内 ( conversation_seq > cursor ] 按序的至多 limit 条消息。
	ListByConversation(ctx context.Context, businessID uint64, conversationID string, cursor int64, limit int) ([]Message, error)
}

var _ MessageDao = (*DefaultMessageDao)(nil)

type DefaultMessageDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultMessageDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultMessageDao {
	return &DefaultMessageDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultMessageDao) SaveInTx(ctx context.Context, tx *gorm.DB, tbName string, message Message) error {
	return tx.WithContext(ctx).Table(tbName).Model(&Message{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colBusinessID}, {Name: colMessageID}},
			DoNothing: true,
		}).
		Create(&message).Error
}

func (d *DefaultMessageDao) ListByConversation(
	ctx context.Context,
	businessID uint64,
	conversationID string,
	cursor int64,
	limit int,
) ([]Message, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewStringSharder(conversationID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from conversation id [ %s ]: %w", conversationID, err)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	messages := make([]Message, 0, limit)
	err = db.WithContext(ctx).Table(dst.TB).Model(&Message{}).
		Where("business_id = ?", businessID).
		Where("conversation_id = ?", conversationID).
		Where("conversation_seq > ?", cursor).
		Order("conversation_seq ASC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

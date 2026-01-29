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

// ConversationView 用户会话视图表。
type ConversationView struct {
	ID uint64 `gorm:"column:id"`

	UserID        uint64 `gorm:"column:user_id"`
	TargetID      uint64 `gorm:"column:target_id"`
	TargetInfoVer int    `gorm:"column:target_info_ver"`

	ConversationType   string `gorm:"column:conversation_type"`
	ConversationName   string `gorm:"column:conversation_name"`
	ConversationRemark string `gorm:"column:conversation_remark"`
	ConversationAvatar string `gorm:"column:conversation_avatar"`

	IsMuted   bool `gorm:"column:is_muted"`
	IsPinned  bool `gorm:"column:is_pinned"`
	IsHidden  bool `gorm:"column:is_hidden"`
	IsStarred bool `gorm:"column:is_starred"`

	LastMessageID         uint64 `gorm:"column:last_message_id"`
	LastMessageType       string `gorm:"column:last_message_type"`
	LastMessageContent    string `gorm:"column:last_message_content"`
	LastMessageSenderID   uint64 `gorm:"column:last_message_sender_id"`
	LastMessageSenderName string `gorm:"column:last_message_sender_name"`
	LastMessageTime       int64  `gorm:"column:last_message_time"`

	UnreadCount  int `gorm:"column:unread_count"`
	MentionCount int `gorm:"column:mention_count"`

	OpenedAt int64 `gorm:"column:opened_at"`
	ClosedAt int64 `gorm:"column:closed_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

// ConversationReverseIndex 会话反向索引表。
type ConversationReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID   uint64 `gorm:"column:user_id"`
	TargetID uint64 `gorm:"column:target_id"`

	ConversationType string `gorm:"column:conversation_type"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type ConversationDao interface {
	// CreateView 创建用户会话视图。
	CreateView(ctx context.Context, conversation ConversationView) error

	ListViewByUserID(ctx context.Context, userID uint64) ([]ConversationView, error)
}

var _ ConversationDao = (*DefaultConversationDao)(nil)

type DefaultConversationDao struct {
	dbs                     *xsync.Map[string, *gorm.DB]
	viewShardHelper         *sharding.ShardHelper
	reverseIndexShardHelper *sharding.ShardHelper
}

func NewDefaultConversationDao(
	dbs *xsync.Map[string, *gorm.DB],
	viewShardHelper, reverseIndexShardHelper *sharding.ShardHelper,
) *DefaultConversationDao {
	return &DefaultConversationDao{
		dbs:                     dbs,
		viewShardHelper:         viewShardHelper,
		reverseIndexShardHelper: reverseIndexShardHelper,
	}
}

func (d *DefaultConversationDao) CreateView(ctx context.Context, conversation ConversationView) error {
	sharder := sharding.NewSingleIDSharder(conversation.UserID)

	id, dst, err := d.viewShardHelper.NextIDAndShard(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", conversation.UserID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 开启事务。
	// 这里需要写入多个数据表:
	// 	- conversation_view
	// 	- conversation_reverse_index
	// 这 2 张表都是以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作。
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 写入 conversation_view 表。
		conversation.ID = id
		conversation.CreatedAt = now
		conversation.UpdatedAt = now
		err = tx.WithContext(ctx).Table(dst.TB).Model(&ConversationView{}).
			Clauses(clause.OnConflict{
				// 唯一索引 ( user_id, target_id, conversation_type ) 冲突时，不进行任何操作。
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "target_id"},
					{Name: "conversation_type"},
				},
				DoNothing: true,
			}).
			Create(&conversation).Error
		if err != nil {
			return fmt.Errorf("failed to save user conversation view: %w", err)
		}

		// 2. 写入 conversation_reverse_index 表。
		reverseIndexDst, err := d.reverseIndexShardHelper.DstFromSharder(sharder)
		if err != nil {
			return fmt.Errorf("failed to get shard destination from user id [ %d ]", conversation.UserID)
		}
		err = tx.WithContext(ctx).Table(reverseIndexDst.TB).Model(&ConversationReverseIndex{}).
			Clauses(clause.OnConflict{
				// 唯一索引 ( user_id, target_id, conversation_type ) 冲突时，不进行任何操作。
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "target_id"},
					{Name: "conversation_type"},
				},
				DoNothing: true,
			}).
			Create(&ConversationReverseIndex{
				ID: id,

				UserID:   conversation.UserID,
				TargetID: conversation.TargetID,

				ConversationType: conversation.ConversationType,

				CreatedAt: now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to save conversation reverse index: %w", err)
		}

		return nil
	})
	return err
}

func (d *DefaultConversationDao) ListViewByUserID(ctx context.Context, userID uint64) ([]ConversationView, error) {
	dst, err := d.viewShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var conversations []ConversationView
	err = db.WithContext(ctx).Table(dst.TB).Model(&ConversationView{}).
		Where("user_id = ?", userID).
		Where("closed_at = ?", 0).
		Order("last_message_time DESC").
		Find(&conversations).Error
	return conversations, err
}

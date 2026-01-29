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

// Channel 聊天频道表 ( 用来表示聊天会话 )。
type Channel struct {
	ID uint64 `gorm:"column:id"`

	UserID        uint64 `gorm:"column:user_id"`
	TargetID      uint64 `gorm:"column:target_id"`
	TargetInfoVer int    `gorm:"column:target_info_ver"`

	ChannelType   string `gorm:"column:channel_type"`
	ChannelName   string `gorm:"column:channel_name"`
	ChannelRemark string `gorm:"column:channel_remark"`
	ChannelAvatar string `gorm:"column:channel_avatar"`

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

// ChannelReverseIndex 频道反向索引表。
type ChannelReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID   uint64 `gorm:"column:user_id"`
	TargetID uint64 `gorm:"column:target_id"`

	ChannelType string `gorm:"column:channel_type"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type ChannelDao interface {
	Create(ctx context.Context, channel Channel) error

	ListByUserID(ctx context.Context, userID uint64) ([]Channel, error)
}

var _ ChannelDao = (*DefaultChannelDao)(nil)

type DefaultChannelDao struct {
	dbs            *xsync.Map[string, *gorm.DB]
	shardHelper    *sharding.ShardHelper
	criShardHelper *sharding.ShardHelper
}

func NewDefaultChannelDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper, criShardHelper *sharding.ShardHelper,
) *DefaultChannelDao {
	return &DefaultChannelDao{
		dbs:            dbs,
		shardHelper:    shardHelper,
		criShardHelper: criShardHelper,
	}
}

func (d *DefaultChannelDao) Create(ctx context.Context, channel Channel) error {
	sharder := sharding.NewSingleIDSharder(channel.UserID)

	id, dst, err := d.shardHelper.NextIDAndShard(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", channel.UserID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 开启事务。
	// 这里需要写入多个数据表:
	// 	- channel_view
	// 	- channel_reverse_index
	// 这 2 张表都是以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作。
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 写入 channel 表。
		channel.ID = id
		channel.CreatedAt = now
		channel.UpdatedAt = now
		err = tx.WithContext(ctx).Table(dst.TB).Model(&Channel{}).
			Clauses(clause.OnConflict{
				// 唯一索引 ( user_id, target_id, channel_type ) 冲突时，不进行任何操作。
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "target_id"},
					{Name: "channel_type"},
				},
				DoNothing: true,
			}).
			Create(&channel).Error
		if err != nil {
			return fmt.Errorf("failed to save user channel view: %w", err)
		}

		// 2. 写入 channel_reverse_index 表。
		reverseIndexDst, err := d.criShardHelper.DstFromSharder(sharder)
		if err != nil {
			return fmt.Errorf("failed to get shard destination from user id [ %d ]", channel.UserID)
		}
		err = tx.WithContext(ctx).Table(reverseIndexDst.TB).Model(&ChannelReverseIndex{}).
			Clauses(clause.OnConflict{
				// 唯一索引 ( user_id, target_id, channel_type ) 冲突时，不进行任何操作。
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "target_id"},
					{Name: "channel_type"},
				},
				DoNothing: true,
			}).
			Create(&ChannelReverseIndex{
				ID: id,

				UserID:   channel.UserID,
				TargetID: channel.TargetID,

				ChannelType: channel.ChannelType,

				CreatedAt: now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to save conversation reverse index: %w", err)
		}

		return nil
	})
	return err
}

func (d *DefaultChannelDao) ListByUserID(ctx context.Context, userID uint64) ([]Channel, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var channels []Channel
	err = db.WithContext(ctx).Table(dst.TB).Model(&Channel{}).
		Where("user_id = ?", userID).
		Where("closed_at = ?", 0).
		Order("last_message_time DESC").
		Find(&channels).Error
	return channels, err
}

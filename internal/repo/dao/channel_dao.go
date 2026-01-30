package dao

import (
	"context"
	"fmt"

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

type ChannelDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, channel Channel) error

	ListByUserID(ctx context.Context, userID uint64) ([]Channel, error)
}

var _ ChannelDao = (*DefaultChannelDao)(nil)

type DefaultChannelDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultChannelDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultChannelDao {
	return &DefaultChannelDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultChannelDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, channel Channel) error {
	return tx.WithContext(ctx).Table(tbName).Model(&Channel{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "target_id"}, {Name: "channel_type"}},
			DoNothing: true,
		}).
		Create(&channel).Error
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

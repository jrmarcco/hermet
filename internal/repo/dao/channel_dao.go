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

	IsMuted  bool `gorm:"column:is_muted"`
	IsPinned bool `gorm:"column:is_pinned"`

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

// ChannelLastMessage 会话最后一条消息的快照 ( 用于会话列表展示 )。
type ChannelLastMessage struct {
	TargetID    uint64
	ChannelType string

	LastMessageID       uint64
	LastMessageType     string
	LastMessageContent  string
	LastMessageSenderID uint64
	LastMessageTime     int64

	// IncrementUnread 为 true 时未读数 +1 ( 接收方 )，发送方自身不递增。
	IncrementUnread bool
}

type ChannelDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, channel Channel) error

	ListByUserID(ctx context.Context, userID uint64) ([]Channel, error)

	// UpdateLastMessageInTx 事务内更新该用户指定会话的最后消息快照
	// ( 会话行由 CDC 链路异步创建，行不存在时静默跳过 )。
	UpdateLastMessageInTx(ctx context.Context, tx *gorm.DB, tbName string, userID uint64, snapshot ChannelLastMessage) error

	// UpdateLastMessageForUsersInTx 事务内批量更新一组用户在同一会话的最后消息快照 ( 群聊扇出用 )。
	// incrementUnread 覆盖 snapshot.IncrementUnread: 调用方按发送者/接收者分组控制未读数。
	UpdateLastMessageForUsersInTx(ctx context.Context, tx *gorm.DB, tbName string, userIDs []uint64, snapshot ChannelLastMessage, incrementUnread bool) error
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
			Columns:   []clause.Column{{Name: colUserID}, {Name: "target_id"}, {Name: "channel_type"}},
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

func (d *DefaultChannelDao) UpdateLastMessageInTx(ctx context.Context, tx *gorm.DB, tbName string, userID uint64, snapshot ChannelLastMessage) error {
	updates := map[string]any{
		"last_message_id":        snapshot.LastMessageID,
		"last_message_type":      snapshot.LastMessageType,
		"last_message_content":   snapshot.LastMessageContent,
		"last_message_sender_id": snapshot.LastMessageSenderID,
		"last_message_time":      snapshot.LastMessageTime,
		colUpdatedAt:             snapshot.LastMessageTime,
	}
	if snapshot.IncrementUnread {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}

	// 会话由 CDC 链路异步创建，消息先到时行可能尚不存在，静默跳过。
	return tx.WithContext(ctx).Table(tbName).Model(&Channel{}).
		Where("user_id = ?", userID).
		Where("target_id = ?", snapshot.TargetID).
		Where("channel_type = ?", snapshot.ChannelType).
		Updates(updates).Error
}

func (d *DefaultChannelDao) UpdateLastMessageForUsersInTx(
	ctx context.Context,
	tx *gorm.DB,
	tbName string,
	userIDs []uint64,
	snapshot ChannelLastMessage,
	incrementUnread bool,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	updates := map[string]any{
		"last_message_id":        snapshot.LastMessageID,
		"last_message_type":      snapshot.LastMessageType,
		"last_message_content":   snapshot.LastMessageContent,
		"last_message_sender_id": snapshot.LastMessageSenderID,
		"last_message_time":      snapshot.LastMessageTime,
		colUpdatedAt:             snapshot.LastMessageTime,
	}
	if incrementUnread {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}

	// 会话由 CDC 链路异步创建，消息先到时行可能尚不存在，静默跳过。
	return tx.WithContext(ctx).Table(tbName).Model(&Channel{}).
		Where("user_id = ANY(?)", toInt64Slice(userIDs)).
		Where("target_id = ?", snapshot.TargetID).
		Where("channel_type = ?", snapshot.ChannelType).
		Updates(updates).Error
}

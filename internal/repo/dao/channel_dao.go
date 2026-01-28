package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const channelMemberBatchSize = 500

// Channel 频道表。
type Channel struct {
	ID uint64 `gorm:"column:id"`

	ChannelName        string `gorm:"column:channel_name"`
	ChannelAvatar      string `gorm:"column:channel_avatar"`
	ChannelStatus      string `gorm:"column:channel_status"`
	ChannelMemberCount int    `gorm:"column:channel_member_count"`

	InfoVer int `gorm:"column:info_ver"`

	LastMessageAt int64 `gorm:"column:last_message_at"`

	CreatorID uint64 `gorm:"column:creator_id"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

// ChannelMember 频道成员表。
type ChannelMember struct {
	ID uint64 `gorm:"column:id"`

	ChannelID uint64 `gorm:"column:channel_id"`
	UserID    uint64 `gorm:"column:user_id"`

	MemberRole string `gorm:"column:member_role"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ChannelMemberView struct {
	ID uint64 `gorm:"column:id"`

	ChannelID uint64 `gorm:"column:channel_id"`
	UserID    uint64 `gorm:"column:user_id"`

	MemberRole string `gorm:"column:member_role"`

	MemberNickname string `gorm:"column:member_nickname"`
	MemberAvatar   string `gorm:"column:member_avatar"`
	MemberGender   string `gorm:"column:member_gender"`
	MemberRemark   string `gorm:"column:member_remark"`
	MemberInfoVer  int    `gorm:"column:member_info_ver"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ChannelDao interface {
	// Create 创建频道。
	Create(ctx context.Context, channel Channel) (Channel, error)
	// AddMembers 添加成员。
	AddMembers(ctx context.Context, channelID uint64, memberIDs []uint64) error

	// UpdateStatus 更新频道状态。
	UpdateStatus(ctx context.Context, id uint64, status string) error
}

var _ ChannelDao = (*DefaultChannelDao)(nil)

type DefaultChannelDao struct {
	dbs                           *xsync.Map[string, *gorm.DB]
	shardHelper                   *sharding.ShardHelper
	memberShardHelper             *sharding.ShardHelper
	memberViewShardHelper         *sharding.ShardHelper
	memberReverseIndexShardHelper *sharding.ShardHelper
}

func NewDefaultChannelDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper, memberShardHelper, memberViewShardHelper, memberReverseIndexShardHelper *sharding.ShardHelper,
) *DefaultChannelDao {
	return &DefaultChannelDao{
		dbs:                           dbs,
		shardHelper:                   shardHelper,
		memberShardHelper:             memberShardHelper,
		memberViewShardHelper:         memberViewShardHelper,
		memberReverseIndexShardHelper: memberReverseIndexShardHelper,
	}
}

func (d *DefaultChannelDao) Create(ctx context.Context, channel Channel) (Channel, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewStringSharder(channel.ChannelName))
	if err != nil {
		return Channel{}, fmt.Errorf("failed to get shard destination from channel name [ %s ]", channel.ChannelName)
	}

	now := time.Now().UnixMilli()

	channel.ID = id
	channel.CreatedAt = now
	channel.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return Channel{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&Channel{}).Create(&channel).Error
	return channel, err
}

func (d *DefaultChannelDao) AddMembers(ctx context.Context, channelID uint64, memberIDs []uint64) error {
	sharder := sharding.NewSingleIDSharder(channelID)
	dst, err := d.memberShardHelper.DstFromSharder(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from channel id [ %d ]", channelID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()
	members := make([]ChannelMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		id, err := d.memberShardHelper.NextID(sharder)
		if err != nil {
			return fmt.Errorf("failed to generate id for member [ %d ]: %w", memberID, err)
		}

		members = append(members, ChannelMember{
			ID: id,

			ChannelID: channelID,
			UserID:    memberID,

			MemberRole: string(domain.ChannelMemberRoleMember),

			JoinedAt: now,
			LeftAt:   0,

			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 批量写入频道成员表。
	return db.WithContext(ctx).Table(dst.TB).Model(&ChannelMember{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "channel_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).
		CreateInBatches(&members, channelMemberBatchSize).Error
}

func (d *DefaultChannelDao) UpdateStatus(ctx context.Context, id uint64, status string) error {
	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	return db.WithContext(ctx).Table(dst.TB).Model(&Channel{}).
		Where("id = ?", id).
		Update("channel_status", status).Error
}

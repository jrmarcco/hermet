package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

const groupMemberBatchSize = 500

// ChatGroup 聊天组表 ( 聊天群 )。
type ChatGroup struct {
	ID uint64 `gorm:"column:id"`

	GroupName   string `gorm:"column:group_name"`
	GroupAvatar string `gorm:"column:group_avatar"`
	GroupStatus string `gorm:"column:group_status"`

	InfoVer int `gorm:"column:info_ver"`

	LastMessageAt int64 `gorm:"column:last_message_at"`

	CreatorID uint64 `gorm:"column:creator_id"`

	DeletedAt int64 `gorm:"column:deleted_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type GroupDao interface {
	// Create 创建聊天群。
	Create(ctx context.Context, group ChatGroup) (ChatGroup, error)

	// UpdateStatus 更新聊天群状态。
	UpdateStatus(ctx context.Context, id uint64, status string) error

	FindByID(ctx context.Context, id uint64) (ChatGroup, error)
}

var _ GroupDao = (*DefaultGroupDao)(nil)

type DefaultGroupDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultGroupDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultGroupDao {
	return &DefaultGroupDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultGroupDao) Create(ctx context.Context, group ChatGroup) (ChatGroup, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewStringSharder(group.GroupName))
	if err != nil {
		return ChatGroup{}, fmt.Errorf("failed to get shard destination from group name [ %s ]", group.GroupName)
	}

	now := time.Now().UnixMilli()

	group.ID = id
	group.CreatedAt = now
	group.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return ChatGroup{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&ChatGroup{}).Create(&group).Error
	if err != nil {
		return ChatGroup{}, fmt.Errorf("failed to create group: %w", err)
	}
	return group, nil
}

func (d *DefaultGroupDao) UpdateStatus(ctx context.Context, id uint64, status string) error {
	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	return db.WithContext(ctx).Table(dst.TB).Model(&ChatGroup{}).
		Where("id = ?", id).
		Update("group_status", status).Error
}

func (d *DefaultGroupDao) FindByID(ctx context.Context, id uint64) (ChatGroup, error) {
	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return ChatGroup{}, fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return ChatGroup{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var group ChatGroup
	err = db.WithContext(ctx).Table(dst.TB).Model(&ChatGroup{}).
		Where("id = ?", id).
		Where("deleted_at = ?", 0).
		First(&group).Error
	return group, err
}

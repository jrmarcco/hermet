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

const groupMemberBatchSize = 500

// ChatGroup 聊天组表 ( 聊天群 )。
type ChatGroup struct {
	ID uint64 `gorm:"column:id"`

	GroupName        string `gorm:"column:group_name"`
	GroupAvatar      string `gorm:"column:group_avatar"`
	GroupStatus      string `gorm:"column:group_status"`
	GroupMemberCount int    `gorm:"column:group_member_count"`

	InfoVer int `gorm:"column:info_ver"`

	LastMessageAt int64 `gorm:"column:last_message_at"`

	CreatorID uint64 `gorm:"column:creator_id"`

	DeletedAt int64 `gorm:"column:deleted_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

// GroupMember 聊天群成员表。
type GroupMember struct {
	ID uint64 `gorm:"column:id"`

	GroupID uint64 `gorm:"column:group_id"`
	UserID  uint64 `gorm:"column:user_id"`

	MemberRole string `gorm:"column:member_role"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type GroupMemberView struct {
	ID uint64 `gorm:"column:id"`

	GroupID uint64 `gorm:"column:group_id"`
	UserID  uint64 `gorm:"column:user_id"`

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

type GroupDao interface {
	// Create 创建聊天群。
	Create(ctx context.Context, group ChatGroup) (ChatGroup, error)
	// CreateMemberView 创建聊天群成员视图。
	CreateMemberView(ctx context.Context, gmv GroupMemberView) (GroupMemberView, error)
	// AddMembers 添加成员。
	AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error

	// UpdateStatus 更新聊天群状态。
	UpdateStatus(ctx context.Context, id uint64, status string) error

	FindByID(ctx context.Context, id uint64) (ChatGroup, error)
}

var _ GroupDao = (*DefaultGroupDao)(nil)

type DefaultGroupDao struct {
	dbs                           *xsync.Map[string, *gorm.DB]
	shardHelper                   *sharding.ShardHelper
	memberShardHelper             *sharding.ShardHelper
	memberViewShardHelper         *sharding.ShardHelper
	memberReverseIndexShardHelper *sharding.ShardHelper
}

func NewDefaultGroupDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper, memberShardHelper, memberViewShardHelper, memberReverseIndexShardHelper *sharding.ShardHelper,
) *DefaultGroupDao {
	return &DefaultGroupDao{
		dbs:                           dbs,
		shardHelper:                   shardHelper,
		memberShardHelper:             memberShardHelper,
		memberViewShardHelper:         memberViewShardHelper,
		memberReverseIndexShardHelper: memberReverseIndexShardHelper,
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
	return group, err
}

func (d *DefaultGroupDao) CreateMemberView(ctx context.Context, gmv GroupMemberView) (GroupMemberView, error) {
	dst, err := d.memberViewShardHelper.DstFromID(gmv.ID)
	if err != nil {
		return GroupMemberView{}, fmt.Errorf("failed to get shard destination from id [ %d ]", gmv.ID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return GroupMemberView{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	gmv.CreatedAt = now
	gmv.UpdatedAt = now

	err = db.WithContext(ctx).Table(dst.TB).Model(&GroupMemberView{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).
		Create(&gmv).Error
	return gmv, err
}

func (d *DefaultGroupDao) AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error {
	sharder := sharding.NewSingleIDSharder(groupID)
	dst, err := d.memberShardHelper.DstFromSharder(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from group id [ %d ]", groupID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()
	members := make([]GroupMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		id, err := d.memberShardHelper.NextID(sharder)
		if err != nil {
			return fmt.Errorf("failed to generate id for member [ %d ]: %w", memberID, err)
		}

		members = append(members, GroupMember{
			ID: id,

			GroupID: groupID,
			UserID:  memberID,

			MemberRole: string(domain.GroupMemberRoleMember),

			JoinedAt: now,
			LeftAt:   0,

			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 批量写入聊天群成员表。
	return db.WithContext(ctx).Table(dst.TB).Model(&GroupMember{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).
		CreateInBatches(&members, groupMemberBatchSize).Error
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

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

type GroupMemberDao interface {
	// BatchCreate 批量创建聊天群成员。
	BatchCreate(ctx context.Context, groupID uint64, memberIDs []uint64) error
}

var _ GroupMemberDao = (*DefaultGroupMemberDao)(nil)

type DefaultGroupMemberDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultGroupMemberDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultGroupMemberDao {
	return &DefaultGroupMemberDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultGroupMemberDao) BatchCreate(ctx context.Context, groupID uint64, memberIDs []uint64) error {
	// 以 group_id 为分片键获取 Sharder。
	sharder := sharding.NewSingleIDSharder(groupID)

	dst, err := d.shardHelper.DstFromSharder(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from group id [ %d ]", groupID)
	}

	// 加载数据库连接。
	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()
	members := make([]GroupMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		id, err := d.shardHelper.NextID(sharder)
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

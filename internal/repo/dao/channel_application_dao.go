package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// GroupApplication 频道申请表。
// 以 GroupID 为分片键。
type GroupApplication struct {
	ID uint64 `gorm:"column:id"`

	ApplicantID uint64 `gorm:"column:applicant_id"`

	GroupID     uint64 `gorm:"column:group_id"`
	GroupName   string `gorm:"column:group_name"`
	GroupAvatar string `gorm:"column:group_avatar"`

	ApplicationStatus  string `gorm:"column:application_status"`
	ApplicationMessage string `gorm:"column:application_message"`

	ReviewerID uint64 `gorm:"column:reviewer_id"`
	ReviewedAt int64  `gorm:"column:reviewed_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type GroupApplicationDao interface {
	Create(ctx context.Context, ca GroupApplication) (GroupApplication, error)
}

var _ GroupApplicationDao = (*DefaultGroupApplicationDao)(nil)

type DefaultGroupApplicationDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultGroupApplicationDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultGroupApplicationDao {
	return &DefaultGroupApplicationDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultGroupApplicationDao) Create(ctx context.Context, ca GroupApplication) (GroupApplication, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewSingleIDSharder(ca.GroupID))
	if err != nil {
		return GroupApplication{}, fmt.Errorf("failed to get shard destination from group id [ %d ]", ca.GroupID)
	}

	now := time.Now().UnixMilli()

	ca.ID = id
	ca.CreatedAt = now
	ca.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return GroupApplication{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&GroupApplication{}).Create(&ca).Error
	return ca, err
}

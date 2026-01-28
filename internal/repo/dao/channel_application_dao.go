package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// ChannelApplication 频道申请表。
// 以 ChannelID 为分片键。
type ChannelApplication struct {
	ID uint64 `gorm:"column:id"`

	ApplicantID uint64 `gorm:"column:applicant_id"`

	ChannelID     uint64 `gorm:"column:channel_id"`
	ChannelName   string `gorm:"column:channel_name"`
	ChannelAvatar string `gorm:"column:channel_avatar"`

	ApplicationStatus  string `gorm:"column:application_status"`
	ApplicationMessage string `gorm:"column:application_message"`

	ReviewerID uint64 `gorm:"column:reviewer_id"`
	ReviewedAt int64  `gorm:"column:reviewed_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ChannelApplicationDao interface {
	Create(ctx context.Context, ca ChannelApplication) (ChannelApplication, error)
}

var _ ChannelApplicationDao = (*DefaultChannelApplicationDao)(nil)

type DefaultChannelApplicationDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultChannelApplicationDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultChannelApplicationDao {
	return &DefaultChannelApplicationDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultChannelApplicationDao) Create(ctx context.Context, ca ChannelApplication) (ChannelApplication, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewSingleIDSharder(ca.ChannelID))
	if err != nil {
		return ChannelApplication{}, fmt.Errorf("failed to get shard destination from channel id [ %d ]", ca.ChannelID)
	}

	now := time.Now().UnixMilli()

	ca.ID = id
	ca.CreatedAt = now
	ca.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return ChannelApplication{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&ChannelApplication{}).Create(&ca).Error
	return ca, err
}

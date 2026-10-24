package dao

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserGroup struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	GroupID uint64 `gorm:"column:group_id"`

	RemarkName string `gorm:"column:remark_name"`

	IsMuted  bool `gorm:"column:is_muted"`
	IsPinned bool `gorm:"column:is_pinned"`

	JoinedAt int64 `gorm:"column:joined_at"`
	LeftAt   int64 `gorm:"column:left_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type UserGroupDao interface {
	CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ug UserGroup) error

	// ExistsActiveMembership 判断用户是否为群内有效成员 ( left_at = 0 )。
	ExistsActiveMembership(ctx context.Context, userID, groupID uint64) (bool, error)
}

var _ UserGroupDao = (*DefaultUserGroupDao)(nil)

type DefaultUserGroupDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserGroupDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultUserGroupDao {
	return &DefaultUserGroupDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultUserGroupDao) CreateInTx(ctx context.Context, tx *gorm.DB, tbName string, ug UserGroup) error {
	return tx.WithContext(ctx).Table(tbName).Model(&UserGroup{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colUserID}, {Name: colGroupID}},
			DoNothing: true,
		}).
		Create(&ug).Error
}

func (d *DefaultUserGroupDao) ExistsActiveMembership(ctx context.Context, userID, groupID uint64) (bool, error) {
	// 以 user_id 为分片键 ( 单行查询 )。
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return false, fmt.Errorf("failed to get shard destination from user id [ %d ]: %w", userID, err)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return false, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var count int64
	err = db.WithContext(ctx).Table(dst.TB).Model(&UserGroup{}).
		Where(colUserID+" = ?", userID).
		Where(colGroupID+" = ?", groupID).
		Where("left_at = ?", 0).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

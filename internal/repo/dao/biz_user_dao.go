package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// BizUser 业务用户表。
type BizUser struct {
	ID uint64 `gorm:"column:id"`

	// 账号信息。
	Email  string `gorm:"column:email"`
	Mobile string `gorm:"column:mobile"`
	Passwd string `gorm:"column:passwd"`

	// 个人信息。
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	Gender   string `gorm:"column:gender"`
	Region   string `gorm:"column:region"`
	Birthday int64  `gorm:"column:birthday"`
	Tagline  string `gorm:"column:tagline"`

	// 版本控制。
	InfoVer int `gorm:"column:info_ver"`

	// 状态管理。
	UserStatus string `gorm:"column:user_status"`
	DeletedAt  int64  `gorm:"column:deleted_at"`

	// 时间戳。
	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

//go:generate mockgen -source=biz_user_dao.go -destination=mock/biz_user_dao.mock.go -package=daomock -typed BizUserDao

type BizUserDao interface {
	Create(ctx context.Context, user BizUser) (BizUser, error)

	FindByID(ctx context.Context, id uint64) (BizUser, error)
	FindByEmail(ctx context.Context, email string) (BizUser, error)
}

var _ BizUserDao = (*DefaultBizUserDao)(nil)

type DefaultBizUserDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultBizUserDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultBizUserDao {
	return &DefaultBizUserDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultBizUserDao) Create(ctx context.Context, user BizUser) (BizUser, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewStringSharder(user.Email))
	if err != nil {
		return BizUser{}, fmt.Errorf("failed to get shard destination from email [ %s ]", user.Email)
	}

	now := time.Now().UnixMilli()

	user.ID = id
	user.CreatedAt = now
	user.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return BizUser{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&BizUser{}).Create(&user).Error
	return user, err
}

func (d *DefaultBizUserDao) FindByID(ctx context.Context, id uint64) (BizUser, error) {
	var user BizUser

	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return BizUser{}, fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return BizUser{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&BizUser{}).
		Where("id = ?", id).
		Where("deleted_at = 0").
		First(&user).Error
	return user, err
}

func (d *DefaultBizUserDao) FindByEmail(ctx context.Context, email string) (BizUser, error) {
	var user BizUser

	dst, err := d.shardHelper.DstFromSharder(sharding.NewStringSharder(email))
	if err != nil {
		return BizUser{}, fmt.Errorf("failed to get shard destination from email [ %s ]", email)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return BizUser{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&BizUser{}).
		Where("email = ?", email).
		Where("deleted_at = 0").
		First(&user).Error
	return user, err
}

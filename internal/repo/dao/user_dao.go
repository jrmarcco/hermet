package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
)

// BizUser 业务用户表。
type BizUser struct {
	ID uint64 `gorm:"column:id"`

	// 账号信息。
	Email  string `gorm:"column:email"`
	Mobile string `gorm:"column:mobile"`
	Passwd string `gorm:"column:passwd"` //nolint:gosec // Database model field for user password hash

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

//go:generate mockgen -source=user_dao.go -destination=mock/user_dao.mock.go -package=daomock -typed UserDao

type UserDao interface {
	Create(ctx context.Context, user BizUser) (BizUser, error)

	FindByID(ctx context.Context, id uint64) (BizUser, error)
	FindByEmail(ctx context.Context, email string) (BizUser, error)
}

var _ UserDao = (*DefaultUserDao)(nil)

type DefaultUserDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultUserDao {
	return &DefaultUserDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultUserDao) Create(ctx context.Context, user BizUser) (BizUser, error) {
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

func (d *DefaultUserDao) FindByID(ctx context.Context, id uint64) (BizUser, error) {
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

func (d *DefaultUserDao) FindByEmail(ctx context.Context, email string) (BizUser, error) {
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

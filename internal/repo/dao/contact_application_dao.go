package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// ContactApplication 联系人申请表。
// 以 TargetID ( user_id ) 为分片键。
type ContactApplication struct {
	ID uint64 `gorm:"column:id"`

	TargetID    uint64 `gorm:"column:target_id"`
	ApplicantID uint64 `gorm:"column:applicant_id"`

	ApplicantName      string `gorm:"column:applicant_name"`
	ApplicantAvatar    string `gorm:"column:applicant_avatar"`
	ApplicationStatus  string `gorm:"column:application_status"`
	ApplicationMessage string `gorm:"column:application_message"`

	Source string `gorm:"column:source"`

	ReviewedAt int64 `gorm:"column:reviewed_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

type ContactApplicationDao interface {
	Create(ctx context.Context, ca ContactApplication) (ContactApplication, error)

	UpdateStatus(ctx context.Context, id uint64, status domain.ApplicationStatus) error

	FindByID(ctx context.Context, id uint64) (ContactApplication, error)
	ListPendingByTargetID(ctx context.Context, targetID uint64) ([]ContactApplication, error)
}

var _ ContactApplicationDao = (*DefaultContactApplicationDao)(nil)

type DefaultContactApplicationDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultContactApplicationDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *DefaultContactApplicationDao {
	return &DefaultContactApplicationDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

func (d *DefaultContactApplicationDao) Create(ctx context.Context, ca ContactApplication) (ContactApplication, error) {
	id, dst, err := d.shardHelper.NextIDAndShard(sharding.NewSingleIDSharder(ca.TargetID))
	if err != nil {
		return ContactApplication{}, err
	}

	now := time.Now().UnixMilli()

	ca.ID = id
	ca.CreatedAt = now
	ca.UpdatedAt = now

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return ContactApplication{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	err = db.WithContext(ctx).Table(dst.TB).Model(&ContactApplication{}).Create(&ca).Error
	return ca, err
}

func (d *DefaultContactApplicationDao) UpdateStatus(ctx context.Context, id uint64, status domain.ApplicationStatus) error {
	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()
	return db.WithContext(ctx).Table(dst.TB).Model(&ContactApplication{}).
		Where("id = ?", id).
		UpdateColumns(map[string]any{
			"application_status": string(status),
			"reviewed_at":        now,
			colUpdatedAt:         now,
		}).Error
}

func (d *DefaultContactApplicationDao) FindByID(ctx context.Context, id uint64) (ContactApplication, error) {
	dst, err := d.shardHelper.DstFromID(id)
	if err != nil {
		return ContactApplication{}, fmt.Errorf("failed to get shard destination from id [ %d ]", id)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return ContactApplication{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var ca ContactApplication
	err = db.WithContext(ctx).Table(dst.TB).Model(&ContactApplication{}).
		Where("id = ?", id).
		First(&ca).Error
	return ca, err
}

func (d *DefaultContactApplicationDao) ListPendingByTargetID(ctx context.Context, targetID uint64) ([]ContactApplication, error) {
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(targetID))
	if err != nil {
		return nil, err
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var cas []ContactApplication
	err = db.WithContext(ctx).Table(dst.TB).Model(&ContactApplication{}).
		Where("target_id = ?", targetID).
		Where("application_status = ?", string(domain.ApplicationStatusPending)).
		Find(&cas).Error
	return cas, err
}

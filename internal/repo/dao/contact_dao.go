package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Contact 【 写入侧 】用户联系人表。
// 以 UserID 为分片键。
type Contact struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	RemarkName string `gorm:"column:remark_name"`
	Source     string `gorm:"column:source"`

	Tags []string `gorm:"column:tags"`

	IsStarred bool `gorm:"column:is_starred"`
	IsBlocked bool `gorm:"column:is_blocked"`

	AddedAt   int64 `gorm:"column:added_at"`
	DeletedAt int64 `gorm:"column:deleted_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

// ContactView 【 读取侧 】用户联系人视图表。
// 以 UserID 为分片键。
type ContactView struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	ContactNickname string `gorm:"column:contact_nickname"`
	ContactAvatar   string `gorm:"column:contact_avatar"`
	ContactGender   string `gorm:"column:contact_gender"`
	ContactInfoVer  int    `gorm:"column:contact_info_ver"`

	RemarkName string   `gorm:"column:remark_name"`
	Tags       []string `gorm:"column:tags"`

	IsStarred bool `gorm:"column:is_starred"`
	IsBlocked bool `gorm:"column:is_blocked"`

	Mobile string `gorm:"column:mobile"`
	Source string `gorm:"column:source"`

	AddedAt   int64 `gorm:"column:added_at"`
	DeletedAt int64 `gorm:"column:deleted_at"`

	CreatedAt int64 `gorm:"column:created_at"`
	UpdatedAt int64 `gorm:"column:updated_at"`
}

// ContactReverseIndex 联系人反向索引表。
type ContactReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	OwnerID uint64 `gorm:"column:owner_id"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type ContactDao interface {
	// Create 创建用户联系人 ( 同时创建视图和反向索引 )。
	Create(ctx context.Context, contact Contact, user BizUser) error

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (Contact, error)
	ListViewByUserID(ctx context.Context, userID uint64) ([]ContactView, error)
}

var _ ContactDao = (*DefaultContactDao)(nil)

type DefaultContactDao struct {
	dbs *xsync.Map[string, *gorm.DB]

	baseShardHelper         *sharding.ShardHelper
	viewShardHelper         *sharding.ShardHelper
	reverseIndexShardHelper *sharding.ShardHelper
}

func NewDefaultContactDao(
	dbs *xsync.Map[string, *gorm.DB],
	baseShardHelper, viewShardHelper, reverseIndexShardHelper *sharding.ShardHelper,
) *DefaultContactDao {
	return &DefaultContactDao{
		dbs: dbs,

		baseShardHelper:         baseShardHelper,
		viewShardHelper:         viewShardHelper,
		reverseIndexShardHelper: reverseIndexShardHelper,
	}
}

func (d *DefaultContactDao) Create(ctx context.Context, contact Contact, user BizUser) error {
	id, dst, err := d.baseShardHelper.NextIDAndShard(sharding.NewSingleIDSharder(contact.UserID))
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", contact.UserID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 开启事务。
	// 这里需要写入多个数据表:
	// 	- contact
	// 	- contact_view
	// 	- contact_reverse_index
	// 这 3 张表都是以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作。
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 写入 contact 表。
		contact.ID = id
		contact.CreatedAt = now
		contact.UpdatedAt = now
		err = tx.WithContext(ctx).Table(dst.TB).Model(&Contact{}).
			// 唯一索引 ( user_id, contact_id ) 冲突时，不进行任何操作。
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "contact_id"}},
				DoNothing: true,
			}).
			Create(&contact).Error
		if err != nil {
			return fmt.Errorf("failed to save user contact: %w", err)
		}

		// 2. 写入 contact_view 表。
		viewDst, err := d.viewShardHelper.DstFromSharder(sharding.NewSingleIDSharder(contact.UserID))
		if err != nil {
			return fmt.Errorf("failed to get shard destination from user id [ %d ]", contact.UserID)
		}
		err = tx.WithContext(ctx).Table(viewDst.TB).Model(&ContactView{}).
			// 唯一索引 ( user_id, contact_id ) 冲突时，不进行任何操作。
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "contact_id"}},
				DoNothing: true,
			}).
			Create(&ContactView{
				ID: id,

				UserID:    contact.UserID,
				ContactID: contact.ContactID,

				ContactNickname: user.Nickname,
				ContactAvatar:   user.Avatar,
				ContactInfoVer:  user.InfoVer,
				ContactGender:   user.Gender,

				Mobile: user.Mobile,
				Source: contact.Source,

				AddedAt:   now,
				DeletedAt: 0,

				CreatedAt: now,
				UpdatedAt: now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to save user contact view: %w", err)
		}

		// 3. 写入 contact_reverse_index 表。
		reverseIndexDst, err := d.reverseIndexShardHelper.DstFromSharder(sharding.NewSingleIDSharder(contact.UserID))
		if err != nil {
			return fmt.Errorf("failed to get shard destination from user id [ %d ]", contact.UserID)
		}
		err = tx.WithContext(ctx).Table(reverseIndexDst.TB).Model(&ContactReverseIndex{}).
			// 唯一索引 ( user_id, owner_id ) 冲突时，不进行任何操作。
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "owner_id"}},
				DoNothing: true,
			}).
			Create(&ContactReverseIndex{
				ID: id,
				// 这里是反向索引 ( userID 为当前用户 / ownerID 为联系人 ID )。
				// 即 owner 的联系人中有当前用户。
				UserID:  contact.UserID,
				OwnerID: contact.ContactID,

				CreatedAt: now,
			}).Error
		if err != nil {
			return fmt.Errorf("failed to save contact reverse index: %w", err)
		}

		return nil
	})
	return err
}

func (d *DefaultContactDao) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (Contact, error) {
	dst, err := d.baseShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return Contact{}, err
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return Contact{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var uc Contact
	err = db.WithContext(ctx).Table(dst.TB).Model(&Contact{}).
		Where("user_id = ?", userID).
		Where("contact_id = ?", contactID).
		Where("deleted_at = ?", 0).
		First(&uc).Error
	if err != nil {
		return Contact{}, err
	}

	return uc, nil
}

func (d *DefaultContactDao) ListViewByUserID(ctx context.Context, userID uint64) ([]ContactView, error) {
	dst, err := d.viewShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var views []ContactView
	err = db.WithContext(ctx).Table(dst.TB).Model(&ContactView{}).
		Where("user_id = ?", userID).
		Where("deleted_at = ?", 0).
		Find(&views).Error
	return views, err
}

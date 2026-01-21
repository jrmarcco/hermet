package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// UserContact 【 写入侧 】用户联系人表。
// 以 UserID 为分片键。
type UserContact struct {
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

// UserContactView 【 读取侧 】用户联系人视图表。
// 以 UserID 为分片键。
type UserContactView struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	ContactNickname string `gorm:"column:contact_nickname"`
	ContactAvatar   string `gorm:"column:contact_avatar"`
	ContactInfoVer  int    `gorm:"column:contact_info_ver"`
	ContactGender   string `gorm:"column:contact_gender"`

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

type ContactReverseIndex struct {
	ID uint64 `gorm:"column:id"`

	UserID  uint64 `gorm:"column:user_id"`
	OwnerID uint64 `gorm:"column:owner_id"`

	CreatedAt int64 `gorm:"column:created_at"`
}

type UserContactDao interface {
	// Create 创建用户联系人 ( 同时创建视图和反向索引 )。
	Create(ctx context.Context, userContact UserContact, contact BizUser) (UserContact, error)

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (UserContact, error)
	ListViewByUserID(ctx context.Context, userID uint64) ([]UserContactView, error)
}

var _ UserContactDao = (*DefaultUserContactDao)(nil)

type DefaultUserContactDao struct {
	dbs *xsync.Map[string, *gorm.DB]

	baseShardHelper         *sharding.ShardHelper
	viewShardHelper         *sharding.ShardHelper
	reverseIndexShardHelper *sharding.ShardHelper
}

func NewDefaultUserContactDao(
	dbs *xsync.Map[string, *gorm.DB],
	baseShardHelper, viewShardHelper, reverseIndexShardHelper *sharding.ShardHelper,
) *DefaultUserContactDao {
	return &DefaultUserContactDao{
		dbs: dbs,

		baseShardHelper:         baseShardHelper,
		viewShardHelper:         viewShardHelper,
		reverseIndexShardHelper: reverseIndexShardHelper,
	}
}

func (d *DefaultUserContactDao) Create(ctx context.Context, userContact UserContact, contact BizUser) (UserContact, error) {
	id, dst, err := d.baseShardHelper.NextIDAndShard(sharding.NewSingleIDSharder(userContact.UserID))
	if err != nil {
		return UserContact{}, fmt.Errorf("failed to get shard destination from user id [ %d ]", userContact.UserID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return UserContact{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 开启事务。
	// 这里需要写入的数据表有:
	// 	- user_contact
	// 	- user_contact_view
	// 	- contact_reverse_index
	// 这 3 张表都是以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作。
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 写入 user_contact 表。
		userContact.ID = id
		userContact.CreatedAt = now
		userContact.UpdatedAt = now
		err = tx.WithContext(ctx).Table(dst.TB).Model(&UserContact{}).Create(&userContact).Error
		if err != nil {
			return fmt.Errorf("failed to save user contact: %w", err)
		}

		// 2. 写入 user_contact_view 表。
		viewDst, err := d.viewShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userContact.UserID))
		if err != nil {
			return fmt.Errorf("failed to get shard destination from user id [ %d ]", userContact.UserID)
		}
		err = tx.WithContext(ctx).Table(viewDst.TB).Model(&UserContactView{}).Create(&UserContactView{
			ID: id,

			UserID:    userContact.UserID,
			ContactID: userContact.ContactID,

			ContactNickname: contact.Nickname,
			ContactAvatar:   contact.Avatar,
			ContactInfoVer:  contact.InfoVer,
			ContactGender:   contact.Gender,

			Mobile: contact.Mobile,
			Source: userContact.Source,

			AddedAt:   now,
			DeletedAt: 0,

			CreatedAt: now,
			UpdatedAt: now,
		}).Error
		if err != nil {
			return fmt.Errorf("failed to save user contact view: %w", err)
		}

		// 3. 写入 contact_reverse_index 表。
		reverseIndexDst, err := d.reverseIndexShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userContact.ContactID))
		if err != nil {
			return fmt.Errorf("failed to get shard destination from contact id [ %d ]", userContact.ContactID)
		}
		err = tx.WithContext(ctx).Table(reverseIndexDst.TB).Model(&ContactReverseIndex{}).Create(&ContactReverseIndex{
			ID: id,
			// 这里是反向索引 ( userID 为当前用户 / ownerID 为联系人 ID )。
			// 即 owner 的联系人中有当前用户。
			UserID:  userContact.UserID,
			OwnerID: userContact.ContactID,

			CreatedAt: now,
		}).Error
		if err != nil {
			return fmt.Errorf("failed to save contact reverse index: %w", err)
		}

		return nil
	})
	return userContact, err
}

func (d *DefaultUserContactDao) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (UserContact, error) {
	dst, err := d.baseShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return UserContact{}, err
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return UserContact{}, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var uc UserContact
	err = db.WithContext(ctx).Table(dst.TB).Model(&UserContact{}).
		Where("user_id = ?", userID).
		Where("contact_id = ?", contactID).
		Where("deleted_at = ?", 0).
		First(&uc).Error
	if err != nil {
		return UserContact{}, err
	}

	return uc, nil
}

func (d *DefaultUserContactDao) ListViewByUserID(ctx context.Context, userID uint64) ([]UserContactView, error) {
	dst, err := d.viewShardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var views []UserContactView
	err = db.WithContext(ctx).Table(dst.TB).Model(&UserContactView{}).
		Where("user_id = ?", userID).
		Where("deleted_at = ?", 0).
		Find(&views).Error
	return views, err
}

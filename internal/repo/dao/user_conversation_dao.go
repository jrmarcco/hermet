package dao

import (
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// UserConversationView 用户会话视图表。
type UserConversationView struct {
	ID uint64 `gorm:"column:id"`

	UserID    uint64 `gorm:"column:user_id"`
	ContactID uint64 `gorm:"column:contact_id"`

	ContactNickname string `gorm:"column:contact_nickname"`
	ContactAvatar   string `gorm:"column:contact_avatar"`
	ContactInfoVer  int    `gorm:"column:contact_info_ver"`
	ContactGender   string `gorm:"column:contact_gender"`

	RemarkName string   `gorm:"column:remark_name"`
	Tags       []string `gorm:"column:tags"`
}

type UserConversationDao interface{}

var _ UserConversationDao = (*DefaultUserConversationDao)(nil)

type DefaultUserConversationDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewDefaultUserConversationDao(
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper *sharding.ShardHelper,
) *DefaultUserConversationDao {
	return &DefaultUserConversationDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

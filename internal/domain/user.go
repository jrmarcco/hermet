package domain

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)

type UserGender string

const (
	UserGenderUnknown UserGender = "unknown"
	UserGenderMale    UserGender = "male"
	UserGenderFemale  UserGender = "female"
)

func (ug UserGender) IsValid() bool {
	switch ug {
	case UserGenderUnknown, UserGenderMale, UserGenderFemale:
		return true
	default:
		return false
	}
}

// BizUser 是业务用户信息。
type BizUser struct {
	ID uint64 // 用户 ID

	// 账号信息。
	Email  string // 邮箱
	Mobile string // 手机号
	Passwd string //nolint:gosec // Database model field for user password hash

	// 个人信息。
	Avatar   string     // 头像
	Nickname string     // 昵称
	Gender   UserGender // 性别
	Region   string     // 地区
	Birthday int64      // 生日
	Tagline  string     // 标签

	// 版本控制。
	InfoVer int // 信息版本

	// 状态管理。
	UserStatus UserStatus // 用户状态
	DeletedAt  int64      // 删除时间戳 ( Unix 毫秒值 )

	// 时间戳。
	CreatedAt int64
	UpdatedAt int64
}

package domain

type ContactSource string

const (
	ContactSourceSearch ContactSource = "search"
	ContactSourceQRCode ContactSource = "qrcode"
	ContactSourceGroup  ContactSource = "group"
)

func (s ContactSource) IsValid() bool {
	switch s {
	case ContactSourceSearch, ContactSourceQRCode, ContactSourceGroup:
		return true
	default:
		return false
	}
}

type Contact struct {
	ID uint64 // 联系人 ID

	UserID    uint64 // 用户 ID
	ContactID uint64 // 联系人 ID

	RemarkName string        // 备注名
	Source     ContactSource // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

	IsStarred bool // 是否星标
	IsBlocked bool // 是否黑名单
	IsMuted   bool // 是否免打扰
	IsPinned  bool // 是否置顶

	AddedAt   int64 // 添加时间戳 ( Unix 毫秒值 )
	DeletedAt int64 // 删除时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

// ContactView 联系人视图。
type ContactView struct {
	ID uint64 // 联系人 ID

	UserID    uint64 // 用户 ID
	ContactID uint64 // 联系人 ID

	ContactNickname string // 联系人昵称
	ContactAvatar   string // 联系人头像
	ContactGender   string // 联系人性别
	ContactMobile   string // 联系人手机号
	ContactInfoVer  int    // 联系人信息版本

	RemarkName string        // 备注名
	Source     ContactSource // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

	IsStarred bool // 是否星标
	IsBlocked bool // 是否黑名单
	IsMuted   bool // 是否免打扰
	IsPinned  bool // 是否置顶

	AddedAt int64 // 添加时间戳 ( Unix 毫秒值 )
}

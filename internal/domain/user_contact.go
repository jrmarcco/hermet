package domain

type UserContactSource string

const (
	UserContactSourceSearch UserContactSource = "search"
	UserContactSourceQRCode UserContactSource = "qrcode"
	UserContactSourceGroup  UserContactSource = "group"
)

func (s UserContactSource) IsValid() bool {
	switch s {
	case UserContactSourceSearch, UserContactSourceQRCode, UserContactSourceGroup:
		return true
	default:
		return false
	}
}

type UserContact struct {
	ID uint64 `json:"id"`

	UserID    uint64 `json:"userId"`
	ContactID uint64 `json:"contactId"`

	RemarkName string            `json:"remarkName"`
	Source     UserContactSource `json:"source"`
	Tags       []string          `json:"tags"`
	GroupName  string            `json:"groupName"`

	IsStarred bool `json:"isStarred"`
	IsBlocked bool `json:"isBlocked"`

	AddedAt   int64 `json:"addedAt"`
	DeletedAt int64 `json:"deletedAt"`

	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

// UserContactView 用户联系人视图。
type UserContactView struct {
	ID uint64 `json:"id"`

	UserID    uint64 `json:"userId"`
	ContactID uint64 `json:"contactId"`

	ContactNickname string `json:"contactNickname"`
	ContactAvatar   string `json:"contactAvatar"`
	ContactInfoVer  int    `json:"contactInfoVer"`
	ContactGender   string `json:"contactGender"`

	RemarkName string   `json:"remarkName"`
	Tags       []string `json:"tags"`

	IsStarred bool `json:"isStarred"`
	IsBlocked bool `json:"isBlocked"`

	Mobile string            `json:"mobile"`
	Source UserContactSource `json:"source"`

	AddedAt int64 `json:"addedAt"`
}

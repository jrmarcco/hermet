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
	ID uint64 `json:"id"`

	UserID    uint64 `json:"userId"`
	ContactID uint64 `json:"contactId"`

	RemarkName string        `json:"remarkName"`
	Source     ContactSource `json:"source"`
	Tags       []string      `json:"tags"`
	GroupName  string        `json:"groupName"`

	IsStarred bool `json:"isStarred"`
	IsBlocked bool `json:"isBlocked"`

	AddedAt   int64 `json:"addedAt"`
	DeletedAt int64 `json:"deletedAt"`

	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

// ContactView 联系人视图。
type ContactView struct {
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

	Mobile string        `json:"mobile"`
	Source ContactSource `json:"source"`

	AddedAt int64 `json:"addedAt"`
}

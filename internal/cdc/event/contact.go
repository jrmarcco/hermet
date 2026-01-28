package event

import "github.com/jrmarcco/hermet/internal/domain"

//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type ContactCDCEvent struct {
	CDCEvent

	ID uint64 `json:"id"`

	UserID    uint64 `json:"user_id"`
	ContactID uint64 `json:"contact_id"`

	RemarkName string               `json:"remark_name"`
	Source     domain.ContactSource `json:"source"`
	Tags       []string             `json:"tags"`
	GroupName  string               `json:"group_name"`

	IsStarred bool `json:"is_starred"`
	IsBlocked bool `json:"is_blocked"`

	AddedAt   int64 `json:"added_at"`
	DeletedAt int64 `json:"deleted_at"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

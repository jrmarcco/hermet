package event

import "github.com/jrmarcco/hermet/internal/domain"

// ContactReviewCDCEvent 是联系人审批 CDC 事件。
//
//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type ContactReviewCDCEvent struct {
	CDCEvent

	ID          uint64 `json:"id"`
	TargetID    uint64 `json:"target_id"`    // 目标用户 ID ( 被申请的用户 ID )
	ApplicantID uint64 `json:"applicant_id"` // 申请人 ID ( 发起申请的用户 ID )

	ApplicantName      string                   `json:"applicant_name"`
	ApplicantAvatar    string                   `json:"applicant_avatar"`
	ApplicationStatus  domain.ApplicationStatus `json:"application_status"`
	ApplicationMessage string                   `json:"application_message"`

	Source domain.ContactSource `json:"source"`

	ReviewedAt int64 `json:"reviewed_at"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

func (e ContactReviewCDCEvent) IsApproved() bool {
	if !e.IsUpdate() {
		return false
	}
	return e.ApplicationStatus == domain.ApplicationStatusApproved
}

// ContactCDCEvent 是联系人关系建立 CDC 事件。
//
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

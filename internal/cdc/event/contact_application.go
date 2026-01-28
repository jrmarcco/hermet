package event

import "github.com/jrmarcco/hermet/internal/domain"

//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type ContactApplicationCDCEvent struct {
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

func (e ContactApplicationCDCEvent) IsApproved() bool {
	if e.Op != "u" {
		return false
	}
	return e.ApplicationStatus == domain.ApplicationStatusApproved
}

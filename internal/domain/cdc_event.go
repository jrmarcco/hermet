package domain

//nolint:tagliatelle // Debezium 标准字段使用 "__xx" 格式。
type CDCEvent struct {
	Op    string `json:"__op"`
	DB    string `json:"__db"`
	Table string `json:"__table"`
	TSMs  int64  `json:"__ts_ms"`
}

//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type ContactApplicationCDCEvent struct {
	CDCEvent

	ID          uint64 `json:"id"`
	TargetID    uint64 `json:"target_id"`    // 目标用户 ID ( 被申请的用户 ID )
	ApplicantID uint64 `json:"applicant_id"` // 申请人 ID ( 发起申请的用户 ID )

	ApplicantName      string            `json:"applicant_name"`
	ApplicantAvatar    string            `json:"applicant_avatar"`
	ApplicationStatus  ApplicationStatus `json:"application_status"`
	ApplicationMessage string            `json:"application_message"`

	Source ContactSource `json:"source"`

	ReviewedAt int64 `json:"reviewed_at"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

func (e ContactApplicationCDCEvent) IsApproved() bool {
	if e.Op != "u" {
		return false
	}
	return e.ApplicationStatus == ApplicationStatusApproved
}

//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type UserContactCDCEvent struct {
	CDCEvent

	ID uint64 `json:"id"`

	UserID    uint64 `json:"user_id"`
	ContactID uint64 `json:"contact_id"`

	RemarkName string        `json:"remark_name"`
	Source     ContactSource `json:"source"`
	Tags       []string      `json:"tags"`
	GroupName  string        `json:"group_name"`

	IsStarred bool `json:"is_starred"`
	IsBlocked bool `json:"is_blocked"`

	AddedAt   int64 `json:"added_at"`
	DeletedAt int64 `json:"deleted_at"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

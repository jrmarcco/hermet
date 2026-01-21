package domain

type ApplicationStatus string

const (
	ApplicationStatusPending  ApplicationStatus = "pending"
	ApplicationStatusApproved ApplicationStatus = "approved"
	ApplicationStatusRejected ApplicationStatus = "rejected"
)

func (s ApplicationStatus) IsValid() bool {
	switch s {
	case ApplicationStatusPending, ApplicationStatusApproved, ApplicationStatusRejected:
		return true
	default:
		return false
	}
}

type ContactApplication struct {
	ID uint64 `json:"id"`

	TargetID uint64 `json:"targetId"` // 目标用户 ID

	ApplicantID     uint64 `json:"applicantId"`     // 申请人 ID
	ApplicantName   string `json:"applicantName"`   // 申请人昵称
	ApplicantAvatar string `json:"applicantAvatar"` // 申请人头像

	ApplicationStatus  ApplicationStatus `json:"applicationStatus"`
	ApplicationMessage string            `json:"applicationMessage"`

	Source UserContactSource `json:"source"` // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

	ReviewedAt int64 `json:"reviewedAt"` // 审批时间戳 ( Unix 毫秒值 )

	CreatedAt int64 `json:"createdAt"` // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 `json:"updatedAt"` // 更新时间戳 ( Unix 毫秒值 )
}

// ChannelApplication 频道申请 ( 入群申请 )。
type ChannelApplication struct {
	ID uint64 `json:"id"`

	ApplicantID uint64 `json:"applicantId"` // 申请人 ID

	ChannelID     uint64 `json:"chanelId"`      // 频道 ID
	ChannelName   string `json:"channelName"`   // 频道名称
	ChannelAvatar string `json:"channelAvatar"` // 频道头像

	ApplicationStatus  ApplicationStatus `json:"applicationStatus"`
	ApplicationMessage string            `json:"applicationMessage"`

	ReviewerID uint64 `json:"reviewerId"` // 审批人 ID
	ReviewedAt int64  `json:"reviewedAt"` // 审批时间戳 ( Unix 毫秒值 )

	CreatedAt int64 `json:"createdAt"` // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 `json:"updatedAt"` // 更新时间戳 ( Unix 毫秒值 )
}

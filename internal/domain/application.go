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
	ID uint64

	TargetID uint64 // 目标用户 ID

	ApplicantID     uint64 // 申请人 ID
	ApplicantName   string // 申请人昵称
	ApplicantAvatar string // 申请人头像

	ApplicationStatus  ApplicationStatus // 申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
	ApplicationMessage string            // 申请消息

	Source ContactSource // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

	ReviewedAt int64 // 审批时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

// ChannelApplication 频道申请 ( 入群申请 )。
type ChannelApplication struct {
	ID uint64

	ApplicantID uint64 // 申请人 ID

	ChannelID     uint64 // 频道 ID
	ChannelName   string // 频道名称
	ChannelAvatar string // 频道头像

	ApplicationStatus  ApplicationStatus // 申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
	ApplicationMessage string            // 申请消息

	ReviewerID uint64 // 审批人 ID
	ReviewedAt int64  // 审批时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

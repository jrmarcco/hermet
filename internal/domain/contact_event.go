package domain

type ContactApplyEvent struct {
	ApplicantID uint64            `json:"applicantId"` // 申请人 ID
	TargetID    uint64            `json:"targetId"`    // 目标用户 ID
	Message     string            `json:"message"`     // 申请消息
	Source      UserContactSource `json:"source"`      // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
}

type ContactReviewEvent struct {
	ApplicationID uint64            `json:"applicationId"`
	Status        ApplicationStatus `json:"status"`
}

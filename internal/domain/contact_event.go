package domain

type ContactApplyEvent struct {
	ApplicantID uint64        // 申请人 ID
	TargetID    uint64        // 目标用户 ID
	Message     string        // 申请消息
	Source      ContactSource // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
}

type ContactReviewEvent struct {
	ApplicationID uint64            // 申请 ID
	Status        ApplicationStatus // 申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
}

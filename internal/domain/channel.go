package domain

type ChannelType string

const (
	ChannelTypeSingle ChannelType = "single"
	ChannelTypeGroup  ChannelType = "group"
)

// Channel 聊天频道。
type Channel struct {
	ID uint64 // 频道 ID

	UserID        uint64 // 用户 ID
	TargetID      uint64 // 目标 ID ( 用户 ID or 频道 ID )
	TargetInfoVer int    // 【 CQRS 关键字段 】目标信息版本号 ( 用于检测 channel / user 信息变更 )

	ChannelType   ChannelType // 频道类型
	ChannelName   string      // 频道名称
	ChannelRemark string      // 频道备注
	ChannelAvatar string      // 频道头像

	IsMuted   bool // 是否免打扰
	IsPinned  bool // 是否置顶
	IsHidden  bool // 是否隐藏
	IsStarred bool // 是否星标

	LastMessageID         uint64
	LastMessageType       string
	LastMessageContent    string
	LastMessageSenderID   uint64 // 最后消息发送者 ID
	LastMessageSenderName string // 最后消息发送者昵称
	LastMessageTime       int64  // 最后消息时间戳 ( Unix 毫秒值 )

	UnreadCount  int
	MentionCount int

	OpenedAt int64
	ClosedAt int64

	CreatedAt int64
	UpdatedAt int64
}

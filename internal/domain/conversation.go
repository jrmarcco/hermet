package domain

type ConversationType string

const (
	ConversationTypeSingle ConversationType = "single"
	ConversationTypeGroup  ConversationType = "group"
)

// ConversationView 会话视图。
type ConversationView struct {
	ID uint64 // 会话 ID

	UserID        uint64 // 用户 ID
	TargetID      uint64 // 目标用户 ID
	TargetInfoVer int    // 目标用户信息版本

	ConversationType   ConversationType // 会话类型
	ConversationName   string           // 会话名称
	ConversationRemark string           // 会话备注
	ConversationAvatar string           // 会话头像

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

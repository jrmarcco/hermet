package domain

type ConversationType string

const (
	ConversationTypeSingle ConversationType = "single"
	ConversationTypeGroup  ConversationType = "group"
)

// ConversationView 会话视图。
type ConversationView struct {
	ID uint64 `json:"id"`

	UserID        uint64 `json:"userId"`
	TargetID      uint64 `json:"targetId"`
	TargetInfoVer int    `json:"targetInfoVer"`

	ConversationType   ConversationType `json:"conversationType"`
	ConversationName   string           `json:"conversationName"`
	ConversationRemark string           `json:"conversationRemark"`
	ConversationAvatar string           `json:"conversationAvatar"`

	IsMuted   bool `json:"isMuted"`
	IsPinned  bool `json:"isPinned"`
	IsHidden  bool `json:"isHidden"`
	IsStarred bool `json:"isStarred"`

	LastMessageID         uint64 `json:"lastMessageId"`
	LastMessageType       string `json:"lastMessageType"`
	LastMessageContent    string `json:"lastMessageContent"`
	LastMessageSenderID   uint64 `json:"lastMessageSenderId"`
	LastMessageSenderName string `json:"lastMessageSenderName"`
	LastMessageTime       int64  `json:"lastMessageTime"`

	UnreadCount  int `json:"unreadCount"`
	MentionCount int `json:"mentionCount"`

	OpenedAt int64 `json:"openedAt"`
	ClosedAt int64 `json:"closedAt"`

	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

package event

// GroupMemberCDCEvent 是聊天群成员添加 CDC 事件。
//
//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type GroupMemberCDCEvent struct {
	CDCEvent

	ID uint64 `json:"id"`

	UserID  uint64 `json:"user_id"`
	GroupID uint64 `json:"group_id"`

	MemberRole string `json:"member_role"`

	JoinedAt int64 `json:"joined_at"`
	LeftAt   int64 `json:"left_at"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

package domain

// GroupCreateEvent 聊天群创建事件。
type GroupCreateEvent struct {
	Name    string // 聊天群名称
	Avatar  string // 聊天群头像
	Creator uint64 // 创建者 ID

	MemberIDs []uint64 // 所有成员 ID 列表
}

// GroupMemberAddEvent 聊天群成员添加事件。
type GroupMemberAddEvent struct {
	GID         uint64   // 聊天群 ID
	MemberIDs   []uint64 // 添加的成员 ID 列表
	IsLastBatch bool     // 是否是最后一批成员
}

// ChatGroupMemberRmEvent 聊天群成员移除事件。
type ChatGroupMemberRmEvent struct {
	GID       uint64   // 聊天群 ID
	MemberIDs []uint64 // 移除的成员 ID 列表
}

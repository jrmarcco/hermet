package domain

// ChannelCreateEvent 频道创建事件。
type ChannelCreateEvent struct {
	Name    string // 频道名称
	Avatar  string // 频道头像
	Creator uint64 // 创建者 ID

	MemberIDs []uint64 // 所有成员 ID 列表
}

// ChannelMemberAddEvent 频道成员添加事件。
type ChannelMemberAddEvent struct {
	CID       uint64   // 频道 ID
	MemberIDs []uint64 // 添加的成员 ID 列表
	Operator  uint64   // 操作者
}

// ChannelMemberRemoveEvent 频道成员移除事件。
type ChannelMemberRemoveEvent struct {
	CID       uint64
	MemberIDs []uint64 // 移除的成员 ID 列表
	Operator  uint64   // 操作者
}

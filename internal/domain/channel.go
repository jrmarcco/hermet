package domain

type ChannelStatus string

const (
	ChannelStatusCreating ChannelStatus = "creating"
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusFailed   ChannelStatus = "failed"
	ChannelStatusArchived ChannelStatus = "archived"
)

// Channel 表示一个聊天频道。
type Channel struct {
	ID uint64 // 频道 ID

	ChannelName        string        // 频道名称
	ChannelAvatar      string        // 频道头像
	ChannelStatus      ChannelStatus // 频道状态
	ChannelMemberCount int           // 频道成员数量

	InfoVer int // 信息版本

	LastMessageAt int64 // 最后消息时间戳 ( Unix 毫秒值 )

	CreatorID uint64 // 创建者 ID

	DeletedAt int64 // 软删除时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )

	Self ChannelMember // 当前用户在频道中的成员信息
}

type ChannelMemberRole string

const (
	ChannelMemberRoleOwner  ChannelMemberRole = "owner"
	ChannelMemberRoleAdmin  ChannelMemberRole = "admin"
	ChannelMemberRoleMember ChannelMemberRole = "member"
)

// ChannelMember 是频道中的成员。
type ChannelMember struct {
	ID uint64 // 成员 ID

	CID uint64 // 频道 ID
	UID uint64 // 用户 ID

	MemberRole ChannelMemberRole // 成员角色

	JoinedAt int64 // 加入时间戳 ( Unix 毫秒值 )
	LeftAt   int64 // 退出时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

type ChannelMemberView struct {
	ID uint64 // 成员 ID

	CID uint64 // 频道 ID
	UID uint64 // 用户 ID

	MemberRole ChannelMemberRole // 成员角色

	MemberNickname string     // 成员昵称
	MemberAvatar   string     // 成员头像
	MemberGender   UserGender // 成员性别
	MemberRemark   string     // 成员备注
	MemberInfoVer  int        // 成员信息版本

	JoinedAt int64 // 加入时间戳 ( Unix 毫秒值 )
	LeftAt   int64 // 退出时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

// ChannelReadRecord 是频道中消息阅读记录。
// 记录了当前频道阅读到的最后一条消息的 ID。
type ChannelReadRecord struct {
	CID           uint64 // 频道 ID
	LastMessageID uint64 // 最后一条消息 ID
}

// ChannelReadRecords 是用户所有频道的消息阅读记录。
type ChannelReadRecords []ChannelReadRecord

func (crr ChannelReadRecords) GetLastMessageID(cid uint64) uint64 {
	for _, r := range crr {
		if r.CID == cid {
			return r.LastMessageID
		}
	}
	return 0
}

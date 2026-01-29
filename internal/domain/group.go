package domain

type ChatGroupStatus string

const (
	ChatGroupStatusCreating ChatGroupStatus = "creating"
	ChatGroupStatusActive   ChatGroupStatus = "active"
	ChatGroupStatusFailed   ChatGroupStatus = "failed"
	ChatGroupStatusArchived ChatGroupStatus = "archived"
)

// ChatGroup 表示一个聊天群。
type ChatGroup struct {
	ID uint64 // 频道 ID

	GroupName        string          // 频道名称
	GroupAvatar      string          // 频道头像
	GroupStatus      ChatGroupStatus // 频道状态
	GroupMemberCount int             // 频道成员数量

	InfoVer int // 信息版本

	LastMessageAt int64 // 最后消息时间戳 ( Unix 毫秒值 )

	CreatorID uint64 // 创建者 ID

	DeletedAt int64 // 软删除时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )

	Self GroupMember // 当前用户在频道中的成员信息
}

type GroupMemberRole string

const (
	GroupMemberRoleOwner  GroupMemberRole = "owner"
	GroupMemberRoleAdmin  GroupMemberRole = "admin"
	GroupMemberRoleMember GroupMemberRole = "member"
)

// GroupMember 是聊天群中的成员。
type GroupMember struct {
	ID uint64 // 成员 ID

	GID uint64 // 聊天群 ID ( group_id )
	UID uint64 // 用户 ID ( user_id )

	MemberRole GroupMemberRole // 成员角色

	JoinedAt int64 // 加入时间戳 ( Unix 毫秒值 )
	LeftAt   int64 // 退出时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

type GroupMemberView struct {
	ID uint64 // 成员 ID

	GID uint64 // 聊天群 ID ( group_id )
	UID uint64 // 用户 ID ( user_id )

	MemberRole GroupMemberRole // 成员角色

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

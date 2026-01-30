package domain

type UserGroup struct {
	ID uint64 // 主键 ID

	UserID  uint64 // 用户 ID
	GroupID uint64 // 群组 ID

	RemarkName string // 群组备注名

	IsMuted  bool // 是否免打扰
	IsPinned bool // 是否置顶

	JoinedAt int64 // 加入时间戳 ( Unix 毫秒值 )
	LeftAt   int64 // 退出时间戳 ( Unix 毫秒值 )

	CreatedAt int64 // 创建时间戳 ( Unix 毫秒值 )
	UpdatedAt int64 // 更新时间戳 ( Unix 毫秒值 )
}

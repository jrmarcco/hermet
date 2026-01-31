package domain

type OutboxAggregateType string

const (
	OutboxAggregateTypeGm OutboxAggregateType = "group_member"
)

type OutboxEventType string

const (
	OutboxEventTypeGmCreate OutboxEventType = "group_member_create" // 聊天群成员首次加入
	OutboxEventTypeGmUpdate OutboxEventType = "group_member_update" // 聊天群成员重新加入
	OutboxEventTypeGmDelete OutboxEventType = "group_member_delete" // 聊天群成员离开
)

type OutboxEvent struct {
	ID uint64 // 事件 ID

	AggregateID   uint64 // 聚合根 ID
	AggregateType string // 聚合根类型

	EventType string // 事件类型
	Payload   []byte // 事件数据
	Metadata  []byte // 事件元数据

	RetryCount int // 重试次数

	CreatedAt int64 // 创建时间 ( Unix 毫秒值 )
}

type GroupMemberOutboxEvent struct {
	GroupID uint64 // 聊天群 ID
	UserID  uint64 // 用户 ID

	JoinedAt int64 // 加入时间 ( Unix 毫秒值 )
	LeftAt   int64 // 离开时间 ( Unix 毫秒值 )
}

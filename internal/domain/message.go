package domain

type ContentType int32

// Message 对应网关消息中的 body 的具体格式。
type Message struct {
	ID uint64 // 消息 ID

	SID        uint64 // 发送者 ID
	RelationID string // 关系 ID ( 用于唯一标识一个会话 )

	MID uint64 // 这里指的是网关消息的唯一 id

	Content     []byte      // 消息内容
	ContentType ContentType // 消息类型

	SendAt int64 // 消息发送时间 ( 统一使用服务器时间 )
}

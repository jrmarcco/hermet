package domain

type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
)

// Message 对应网关消息中的 body 的具体格式。
type Message struct {
	ID uint64 // 消息 ID

	SID uint64 // Sender ID ( 发送者 ID )
	MID uint64 // Message ID ( 这里指的是网关消息的 ID )

	DestinationID string // Destination ID ( 目的地频道 ID )

	Content     []byte      // 消息内容
	ContentType ContentType // 消息类型

	SendAt int64 // 消息发送时间 ( 统一使用服务器时间 )
}

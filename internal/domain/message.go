package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type ContentType string

const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
)

// Message 对应网关 UPSTREAM 帧 body 的业务格式 ( 前端自定义，后端只透传 )。
//
// 注意: 身份字段以网关盖章的 UpstreamMessage 为准 ( 契约安全前提 )，
// body 内自报的 SID 仅作展示用途，不做权限判断依据。
type Message struct {
	ID string // 消息 ID

	SID uint64 // Sender ID ( 发送者 ID )
	MID uint64 // Message ID ( 这里指的是网关消息的 ID )

	DestinationID   uint64      // Destination ID ( 目的地 ID：用户 ID 或 群 ID )
	DestinationType ChannelType // Destination Type ( 目的地类型 )

	Content     []byte      // 消息内容
	ContentType ContentType // 消息类型

	SendAt int64 // 消息发送时间 ( 统一使用服务器时间 )
}

// SingleChatMsg 待落库的单聊消息 ( 服务层组装 )。
type SingleChatMsg struct {
	BusinessID uint64

	// MessageID 网关 UPSTREAM 帧的 message_id ( 幂等去重键 )。
	MessageID string
	// SnowflakeID 消息正本表主键 ( 雪花 ID )。
	SnowflakeID uint64

	SenderID   uint64
	ReceiverID uint64

	ConversationID string

	Body        []byte
	ContentType string
	// ContentPreview 会话列表展示用的内容摘要 ( 截断至 channel.last_message_content 列宽 )。
	ContentPreview string

	SentAt int64
}

// SingleChatStoreResult 单聊落库结果。
type SingleChatStoreResult struct {
	SenderSeq   int64
	ReceiverSeq int64
}

// GroupChatMsg 待落库的群聊消息 ( 服务层组装 )。
type GroupChatMsg struct {
	BusinessID uint64

	// MessageID 网关 UPSTREAM 帧的 message_id ( 幂等去重键 )。
	MessageID string
	// SnowflakeID 消息正本表主键 ( 雪花 ID )。
	SnowflakeID uint64

	SenderID uint64
	GroupID  uint64

	ConversationID string

	Body        []byte
	ContentType string
	// ContentPreview 会话列表展示用的内容摘要 ( 截断至 channel.last_message_content 列宽 )。
	ContentPreview string

	SentAt int64

	// MemberIDs 当前群内有效成员 ( 含发送者，发送者多端自同步 )。
	MemberIDs []uint64

	// Inline 为 true 时收件箱写扩散内联正本 ( 小群 )，
	// false 时写读扩散指针 ( 大群，客户端需再发起 CONVERSATION_SYNC 拉取正文 )。
	Inline bool
}

// GroupChatStoreResult 群聊落库结果: 各成员收件箱的最新 seq ( 用于投递通知 )。
type GroupChatStoreResult struct {
	MemberSeqs map[uint64]int64
}

// TimelineMessage 会话 timeline 中的一条消息 ( ConversationSync 下发 )。
type TimelineMessage struct {
	ConversationSeq int64

	MessageID      string
	ConversationID string
	SenderID       uint64

	Body        []byte
	ContentType string

	SentAt int64
}

// InboxEntryKind 收件箱条目类型。
type InboxEntryKind int16

const (
	// InboxEntryKindChat 写扩散条目 ( 消息本体内联 )。
	InboxEntryKindChat InboxEntryKind = 1
	// InboxEntryKindPointer 读扩散指针 ( 会话更新 )。
	InboxEntryKindPointer InboxEntryKind = 2
)

// InboxEntry 收件箱条目 ( SyncService 同步链路下发 )。
type InboxEntry struct {
	Seq int64

	Kind InboxEntryKind

	MessageID string

	ConversationID        string
	ConversationLatestSeq int64

	SenderID    uint64
	Body        []byte
	ContentType string
	SentAt      int64
}

// conversation id 前缀。
const (
	conversationPrefixSingle = "s"
	conversationPrefixGroup  = "g"
)

// SingleConversationID 返回单聊会话 ID ( 双方 UID 排序后拼接，保证 A->B 与 B->A 一致 )。
func SingleConversationID(a, b uint64) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%s:%d:%d", conversationPrefixSingle, a, b)
}

// GroupConversationID 返回群聊会话 ID。
func GroupConversationID(groupID uint64) string {
	return fmt.Sprintf("%s:%d", conversationPrefixGroup, groupID)
}

// ParseGroupConversationID 解析群聊会话 ID，非群聊会话 ( 或格式非法 ) 时第二个返回值为 false。
func ParseGroupConversationID(conversationID string) (uint64, bool) {
	rest, ok := strings.CutPrefix(conversationID, conversationPrefixGroup+":")
	if !ok {
		return 0, false
	}

	groupID, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return groupID, true
}

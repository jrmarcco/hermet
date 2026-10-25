package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ContentType string

const (
	ContentTypeText   ContentType = "text"
	ContentTypeImage  ContentType = "image"
	ContentTypeSystem ContentType = "system"
)

// SystemTypeSendFailed 系统回执类型: UPSTREAM 消息业务级发送失败。
const SystemTypeSendFailed = "send_failed"

// SendFailReason 业务级发送失败原因 ( 系统回执 body.reason )。
type SendFailReason string

const (
	// SendFailReasonNotAllowed 目的地存在但无权发送 ( 非好友 / 被屏蔽 / 非群成员 )。
	SendFailReasonNotAllowed SendFailReason = "not_allowed"
	// SendFailReasonInvalidDestination 目的地非法 ( 缺失 / 指向自己 / 未知类型 )。
	SendFailReasonInvalidDestination SendFailReason = "invalid_destination"
	// SendFailReasonInvalidBody body 不是合法的业务消息 JSON。
	SendFailReasonInvalidBody SendFailReason = "invalid_body"
)

// NoticeText 返回面向发送者的回执文案 ( 不透出内部实现细节 )。
func (r SendFailReason) NoticeText() string {
	switch r {
	case SendFailReasonNotAllowed:
		return "消息发送失败: 对方不是你的联系人或你不是群成员"
	case SendFailReasonInvalidDestination:
		return "消息发送失败: 目的地不存在或类型非法"
	case SendFailReasonInvalidBody:
		return "消息发送失败: 消息体格式非法"
	default:
		return "消息发送失败"
	}
}

// SystemNotice 是系统回执的 body 格式 ( contentType = system )。
// 客户端按 system 字段识别，未识别的客户端可退化为按 content 文本渲染。
type SystemNotice struct {
	SystemType string `json:"systemType"`

	// Reason 取 SendFailReason 常量。
	Reason string `json:"reason"`

	// UpstreamID 触发本次回执的 UPSTREAM 帧 message_id，
	// 发送端以此关联本地 pending 消息。
	UpstreamID string `json:"upstreamId"`

	// Content 面向用户的文案。
	Content string `json:"content"`
}

// Message 对应网关 UPSTREAM 帧 body 的业务格式 ( 前端自定义，后端只透传 )。
//
// 注意: 身份字段以网关盖章的 UpstreamMessage 为准 ( 契约安全前提 )，
// body 内自报的 SID 仅作展示用途，不做权限判断依据。
type Message struct {
	ID string `json:"id"` // 消息 ID

	SID uint64 `json:"sid"` // Sender ID ( 发送者 ID )
	MID uint64 `json:"mid"` // Message ID ( 这里指的是网关消息的 ID )

	DestinationID   uint64      `json:"destinationId"`   // Destination ID ( 目的地 ID：用户 ID 或 群 ID )
	DestinationType ChannelType `json:"destinationType"` // Destination Type ( 目的地类型 )

	Content     []byte      `json:"content"`     // 消息内容
	ContentType ContentType `json:"contentType"` // 消息类型

	SendAt int64 `json:"sendAt"` // 消息发送时间 ( 统一使用服务器时间 )
}

// UnmarshalJSON 兼容前端的历史与现状编码:
//
//   - Content 为 []byte 时标准库期望 base64，而业务契约语义是明文字符串
//     ( 见 docs/frontend-api.md §5 )——这里显式按字符串接收。
//   - int64 ID 字段同时接受十进制字符串与数字字面量: ID 超出 JS 安全整数
//     范围时前端只能以字符串承载 ( 服务端响应也已统一字符串化 )，
//     字符串形态的 destinationId 不再被静默丢弃。
func (m *Message) UnmarshalJSON(data []byte) error {
	type alias Message

	aux := struct {
		*alias

		Content *string `json:"content"`

		SID           *flexibleU64 `json:"sid"`
		MID           *flexibleU64 `json:"mid"`
		DestinationID *flexibleU64 `json:"destinationId"`
	}{alias: (*alias)(m)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Content != nil {
		m.Content = []byte(*aux.Content)
	}
	if aux.SID != nil {
		m.SID = uint64(*aux.SID)
	}
	if aux.MID != nil {
		m.MID = uint64(*aux.MID)
	}
	if aux.DestinationID != nil {
		m.DestinationID = uint64(*aux.DestinationID)
	}
	return nil
}

// MarshalJSON 与 UnmarshalJSON 对称: Content 按明文字符串序列化。
//
// 标准库对 []byte 做 base64 编码，与业务契约的明文语义不符
// ( 管理端测试接口会经 Marshal -> UnmarshalJSON 往返 )。
func (m Message) MarshalJSON() ([]byte, error) {
	type alias Message

	return json.Marshal(struct {
		alias
		Content string `json:"content"`
	}{
		alias:   alias(m),
		Content: string(m.Content),
	})
}

// flexibleU64 兼容十进制字符串与数字字面量的 uint64。
type flexibleU64 uint64

func (u *flexibleU64) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*u = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*u = flexibleU64(val)
	return nil
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

// MessageIndexEvent 消息索引事件 ( 消息落库后投递，ES indexer 消费 )。
//
// ES 只是异步查询加速层 ( 可重建、最终一致 ):
//   - 事件丢失可接受 ( reindex 工具可从 PG 全量重建 );
//   - 事件重复可接受 ( ES 以 message_id 作为文档 _id，重复写入幂等 );
//   - 索引链路绝不进入 SYNC 同步主链路。
type MessageIndexEvent struct {
	MessageID string `json:"messageId"`

	BusinessID     uint64 `json:"businessId"`
	ConversationID string `json:"conversationId"`
	SenderID       uint64 `json:"senderId"`

	Body        []byte `json:"body"`
	ContentType string `json:"contentType"`

	SentAt int64 `json:"sentAt"`
}

// MarshalJSON 与 UnmarshalJSON 对称: Body 按明文字符串序列化。
//
// ES 全文检索要求 body 是可分词文本，标准库对 []byte 的 base64 编码
// 会让搜索失效——这里统一按字符串承载 ( 与 Message.Content 语义一致 )。
func (m MessageIndexEvent) MarshalJSON() ([]byte, error) {
	type alias MessageIndexEvent

	return json.Marshal(struct {
		alias
		Body string `json:"body"`
	}{
		alias: alias(m),
		Body:  string(m.Body),
	})
}

// UnmarshalJSON 兼容字符串形式的 body ( 与 Message.UnmarshalJSON 同理 )。
func (m *MessageIndexEvent) UnmarshalJSON(data []byte) error {
	type alias MessageIndexEvent

	aux := struct {
		*alias
		Body *string `json:"body"`
	}{alias: (*alias)(m)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Body != nil {
		m.Body = []byte(*aux.Body)
	}
	return nil
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
	BusinessID uint64
	UserID     uint64

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

// ParseSingleConversationID 解析单聊会话 ID，非单聊会话 ( 或格式非法 ) 时返回 false。
// 返回的双方 UID 与生成时一致 ( a < b 升序 )。
func ParseSingleConversationID(conversationID string) (a, b uint64, ok bool) {
	rest, cut := strings.CutPrefix(conversationID, conversationPrefixSingle+":")
	if !cut {
		return 0, 0, false
	}

	left, right, found := strings.Cut(rest, ":")
	if !found {
		return 0, 0, false
	}

	a, aErr := strconv.ParseUint(left, 10, 64)
	b, bErr := strconv.ParseUint(right, 10, 64)
	if aErr != nil || bErr != nil {
		return 0, 0, false
	}
	return a, b, true
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// channel.last_message_content 列宽 ( scripts/sql/pgsql/02_read_side.sql )。
const lastMessageContentMaxRunes = 512

// MessageService 处理网关投递到 Kafka 上行 topic 的业务消息。
type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, upstreamMsg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	messageRepo repo.MessageRepo
	contactRepo repo.ContactRepo

	// notifyProducer 投递 NotifyMessage 到通知 topic ( 推送侧消费后下发 NOTIFY 帧 )。
	notifyProducer mq.Producer

	idgen idgen.Generator

	logger *zap.Logger
}

func NewDefaultMessageService(
	messageRepo repo.MessageRepo,
	contactRepo repo.ContactRepo,

	notifyProducer mq.Producer,

	gen idgen.Generator,

	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		messageRepo: messageRepo,
		contactRepo: contactRepo,

		notifyProducer: notifyProducer,

		idgen: gen,

		logger: logger,
	}
}

// ProcessUpstreamMsg 处理上游消息 ( 网关盖章后的 UpstreamMessage，protojson 编码 )。
//
// 落库即事实: 消息只有在这里持久化并分配 seq 后才算成立；
// 随后投递通知 ( 只携带 latest_seq，不携带消息本体 )，客户端经 SYNC 拉取增量。
//
// 错误语义:
//   - 解析失败 / 校验不通过: 记录日志后返回 nil ( 毒消息与业务拒绝不重投，避免无限循环 )。
//   - 落库失败: 返回 error，Kafka 按消费语义重投 ( 全链路幂等，重投安全 )。
//   - 通知失败: 仅记录日志 ( 契约允许通知丢失，只影响实时性 )。
func (s *DefaultMessageService) ProcessUpstreamMsg(ctx context.Context, upstreamMsg *xmq.Message) error {
	stamped := &messagev1.UpstreamMessage{}
	if err := protojson.Unmarshal(upstreamMsg.Val, stamped); err != nil {
		s.logger.Error(
			"[message-service] failed to unmarshal upstream message, skip",
			zap.ByteString("val", upstreamMsg.Val),
			zap.Error(err),
		)
		return nil
	}

	// 身份以网关盖章为准，body 内自报的 SID 仅作展示。
	businessID := stamped.GetBusinessId()
	if businessID == 0 {
		businessID = DefaultBusinessID
	}
	senderID := stamped.GetUserId()

	body := &domain.Message{}
	if err := json.Unmarshal(stamped.GetBody(), body); err != nil {
		s.logger.Error(
			"[message-service] failed to unmarshal upstream body, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("sender_id", senderID),
			zap.Error(err),
		)
		return nil
	}

	switch body.DestinationType {
	case domain.ChannelTypeSingle:
		return s.processSingleChat(ctx, stamped, businessID, senderID, body)
	case domain.ChannelTypeGroup:
		// 群聊走读扩散 ( conversation timeline + 指针扇出 )，暂未实现。
		s.logger.Warn(
			"[message-service] group message not supported yet, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("group_id", body.DestinationID),
		)
		return nil
	default:
		s.logger.Error(
			"[message-service] unknown destination type, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.String("destination_type", string(body.DestinationType)),
		)
		return nil
	}
}

func (s *DefaultMessageService) processSingleChat(
	ctx context.Context,
	stamped *messagev1.UpstreamMessage,
	businessID, senderID uint64,
	body *domain.Message,
) error {
	receiverID := body.DestinationID
	if receiverID == 0 || receiverID == senderID {
		s.logger.Error(
			"[message-service] invalid single chat destination, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("sender_id", senderID),
			zap.Uint64("receiver_id", receiverID),
		)
		return nil
	}

	if err := s.validateContact(ctx, senderID, receiverID); err != nil {
		// 业务级失败不通过 UPSTREAM_ACK 表达 ( 网关级回执与业务结果解耦，见契约 )，
		// 后续可作为系统消息走同步链路下发，当前先记录日志。
		s.logger.Warn(
			"[message-service] single chat validation failed, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("sender_id", senderID),
			zap.Uint64("receiver_id", receiverID),
			zap.Error(err),
		)
		return nil
	}

	sentAt := time.Now().UnixMilli()
	conversationID := domain.SingleConversationID(senderID, receiverID)

	snowflakeID, err := s.nextSnowflakeID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to generate message id: %w", err)
	}

	msg := domain.SingleChatMsg{
		BusinessID: businessID,

		MessageID:   stamped.GetMessageId(),
		SnowflakeID: snowflakeID,

		SenderID:   senderID,
		ReceiverID: receiverID,

		ConversationID: conversationID,

		Body:           body.Content,
		ContentType:    string(body.ContentType),
		ContentPreview: contentPreview(body),

		SentAt: sentAt,
	}

	result, err := s.messageRepo.StoreSingleChat(ctx, msg)
	if err != nil {
		// 返回 error 触发消费重投 ( 幂等 )。
		return fmt.Errorf("failed to store single chat message: %w", err)
	}

	// 通知只携带 latest_seq ( 推送即通知，不携带消息本体 )。
	// 分区键 hash(business_id, receiver_id) 保证同一接收者的通知有序。
	s.notify(ctx, stamped.GetMessageId(), businessID, senderID, result.SenderSeq)
	s.notify(ctx, stamped.GetMessageId(), businessID, receiverID, result.ReceiverSeq)

	return nil
}

func (s *DefaultMessageService) validateContact(ctx context.Context, senderID, receiverID uint64) error {
	contact, err := s.contactRepo.FindByUserIDAndContactID(ctx, senderID, receiverID)
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return fmt.Errorf("sender [ %d ] and receiver [ %d ] are not contacts", senderID, receiverID)
		}
		return err
	}

	if contact.DeletedAt != 0 {
		return fmt.Errorf("contact deleted between [ %d ] and [ %d ]", senderID, receiverID)
	}
	if contact.IsBlocked {
		return fmt.Errorf("receiver [ %d ] is blocked by sender [ %d ]", receiverID, senderID)
	}
	return nil
}

func (s *DefaultMessageService) notify(ctx context.Context, messageID string, businessID, receiverID uint64, latestSeq int64) {
	notifyMsg := &messagev1.NotifyMessage{
		MessageId: messageID,

		BusinessId: businessID,
		ReceiverId: receiverID,

		LatestSeq: uint64(latestSeq),
	}

	val, err := protojson.Marshal(notifyMsg)
	if err != nil {
		// proto 序列化失败属于代码缺陷，记录后丢弃该条通知 ( 通知可丢 )。
		s.logger.Error(
			"[message-service] failed to marshal notify message",
			zap.String("message_id", messageID),
			zap.Uint64("receiver_id", receiverID),
			zap.Error(err),
		)
		return
	}

	key := fmt.Appendf(nil, "%d:%d", businessID, receiverID)
	if err := s.notifyProducer.Produce(ctx, key, val); err != nil {
		// 通知失败无需重试 ( 只影响实时性，客户端重连/心跳时主动 SYNC 兜底 )。
		s.logger.Warn(
			"[message-service] failed to produce notify message",
			zap.String("message_id", messageID),
			zap.Uint64("receiver_id", receiverID),
			zap.Error(err),
		)
	}
}

// nextSnowflakeID 生成消息正本表主键 ( 分片值取自 conversation_id 的哈希 )。
func (s *DefaultMessageService) nextSnowflakeID(conversationID string) (uint64, error) {
	shardVal, err := sharding.NewStringSharder(conversationID).ShardVal()
	if err != nil {
		return 0, err
	}
	return s.idgen.NextID(shardVal)
}

// contentPreview 生成会话列表展示用的内容摘要 ( 按 rune 截断至列宽 )。
func contentPreview(body *domain.Message) string {
	preview := string(body.Content)
	if utf8.RuneCountInString(preview) <= lastMessageContentMaxRunes {
		return preview
	}

	runes := []rune(preview)
	return string(runes[:lastMessageContentMaxRunes])
}

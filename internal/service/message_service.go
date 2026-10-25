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
	"github.com/jrmarcco/xgrpc/client"
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

// MessageFanoutConfig 消息扇出配置 ( hermet.message )。
type MessageFanoutConfig struct {
	// InlineThreshold 写扩散/读扩散阈值:
	// 群成员数 <= 阈值收件箱内联正本 ( 一次 SYNC 即可渲染 )，超过只写轻量指针。
	InlineThreshold int

	// BatchSize 群扇出每批成员数 ( 每批一个分片事务，受 statement_timeout 约束 )。
	BatchSize int
}

// notifyGatewayServiceName 网关在服务发现中的注册名 ( gateway/etc grpc.name )。
const notifyGatewayServiceName = "synp-gateway"

// instantNotifyTimeout 即时通知调用超时 ( 尽力而为的补充通道，宁短勿等 )。
const instantNotifyTimeout = 300 * time.Millisecond

type DefaultMessageService struct {
	messageRepo repo.MessageRepo
	contactRepo repo.ContactRepo

	groupMemberRepo repo.GroupMemberRepo
	userGroupRepo   repo.UserGroupRepo

	// notifyProducer 投递 NotifyMessage 到通知 topic ( 推送侧消费后下发 NOTIFY 帧 )。
	notifyProducer mq.Producer
	// notifyClients 网关 NotifyService 客户端 ( 即时通道，尽力而为 )。
	notifyClients *client.Manager[messagev1.NotifyServiceClient]
	// indexProducer 投递索引事件到 ES 索引 topic ( 异步查询加速层，可丢可重 )。
	indexProducer mq.Producer

	fanoutCfg MessageFanoutConfig

	idgen idgen.Generator

	logger *zap.Logger
}

func NewDefaultMessageService(
	messageRepo repo.MessageRepo,
	contactRepo repo.ContactRepo,

	groupMemberRepo repo.GroupMemberRepo,
	userGroupRepo repo.UserGroupRepo,

	notifyProducer mq.Producer,
	notifyClients *client.Manager[messagev1.NotifyServiceClient],
	indexProducer mq.Producer,

	fanoutCfg MessageFanoutConfig,

	gen idgen.Generator,

	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		messageRepo: messageRepo,
		contactRepo: contactRepo,

		groupMemberRepo: groupMemberRepo,
		userGroupRepo:   userGroupRepo,

		notifyProducer: notifyProducer,
		notifyClients:  notifyClients,
		indexProducer:  indexProducer,

		fanoutCfg: fanoutCfg,

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
		return s.processGroupChat(ctx, stamped, businessID, senderID, body)
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

	// 索引事件 ( ES 查询加速层，可丢可重，失败不影响主链路 )。
	s.emitIndexEvent(ctx, msg.BusinessID, msg.MessageID, msg.ConversationID, msg.SenderID, msg.Body, msg.ContentType, msg.SentAt)

	return nil
}

// processGroupChat 处理群聊消息 ( 读扩散为主，小群走写扩散内联 ):
//
//  1. 校验发送者群成员身份 ( user_group，单行查询 )。
//  2. 会话分片事务: 分配 conversation_seq + 写正本 ( 全群一份 )。
//  3. 收件箱扇出: 小群内联正本 / 大群轻量指针，每成员一条。
//  4. 通知扇出: 每成员一条 NotifyMessage ( 仅 latest_seq )。
func (s *DefaultMessageService) processGroupChat(
	ctx context.Context,
	stamped *messagev1.UpstreamMessage,
	businessID, senderID uint64,
	body *domain.Message,
) error {
	groupID := body.DestinationID
	if groupID == 0 {
		s.logger.Error(
			"[message-service] invalid group destination, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("sender_id", senderID),
		)
		return nil
	}

	active, err := s.userGroupRepo.IsActiveMember(ctx, senderID, groupID)
	if err != nil {
		// 存储层错误，重投 ( 幂等 )。
		return fmt.Errorf("failed to check group membership: %w", err)
	}
	if !active {
		s.logger.Warn(
			"[message-service] sender is not an active group member, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("sender_id", senderID),
			zap.Uint64("group_id", groupID),
		)
		return nil
	}

	memberIDs, err := s.groupMemberRepo.ListActiveMemberIDs(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to list group members: %w", err)
	}
	if len(memberIDs) == 0 {
		s.logger.Warn(
			"[message-service] group has no active members, skip",
			zap.String("message_id", stamped.GetMessageId()),
			zap.Uint64("group_id", groupID),
		)
		return nil
	}

	conversationID := domain.GroupConversationID(groupID)

	snowflakeID, err := s.nextSnowflakeID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to generate message id: %w", err)
	}

	msg := domain.GroupChatMsg{
		BusinessID: businessID,

		MessageID:   stamped.GetMessageId(),
		SnowflakeID: snowflakeID,

		SenderID: senderID,
		GroupID:  groupID,

		ConversationID: conversationID,

		Body:           body.Content,
		ContentType:    string(body.ContentType),
		ContentPreview: contentPreview(body),

		SentAt: time.Now().UnixMilli(),

		MemberIDs: memberIDs,

		// 小群写扩散 ( 收件箱内联正本，一次 SYNC 即可渲染 )，
		// 大群读扩散 ( 收件箱只写指针，客户端再发起 CONVERSATION_SYNC 拉取正文 )。
		// 阈值是部署参数，网关与客户端均不感知。
		Inline: len(memberIDs) <= s.fanoutCfg.InlineThreshold,
	}

	result, err := s.messageRepo.StoreGroupChat(ctx, msg, s.fanoutCfg.BatchSize)
	if err != nil {
		// 返回 error 触发消费重投 ( 幂等 )。
		return fmt.Errorf("failed to store group chat message: %w", err)
	}

	// 通知扇出: 每成员一条，只携带 latest_seq。
	// 分区键 hash(business_id, receiver_id) 保证同一接收者的通知有序。
	for memberID, memberSeq := range result.MemberSeqs {
		s.notify(ctx, msg.MessageID, businessID, memberID, memberSeq)
	}

	// 索引事件 ( ES 查询加速层，可丢可重，失败不影响主链路 )。
	s.emitIndexEvent(ctx, msg.BusinessID, msg.MessageID, msg.ConversationID, msg.SenderID, msg.Body, msg.ContentType, msg.SentAt)

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

	// 即时通道 ( 网关 NotifyService ): 命中本节点在线连接时毫秒级下发，
	// 未命中/失败均无碍——下方 Kafka 广播链路兜底。通知可重复，客户端按 seq 幂等。
	s.instantNotify(ctx, notifyMsg)

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

// instantNotify 经网关 NotifyService 即时通知 ( 尽力而为，短超时 )。
// 网关节点只通知本节点在线的连接，USER_OFFLINE / 超时 / 错误一律静默。
func (s *DefaultMessageService) instantNotify(ctx context.Context, notifyMsg *messagev1.NotifyMessage) {
	if s.notifyClients == nil {
		return
	}

	grpcClient, err := s.notifyClients.Get(notifyGatewayServiceName)
	if err != nil {
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, instantNotifyTimeout)
	defer cancel()

	// 任何结果都不重试: 通知只影响实时性。
	_, _ = grpcClient.Notify(callCtx, &messagev1.NotifyRequest{Message: notifyMsg})
}

// nextSnowflakeID 生成消息正本表主键 ( 分片值取自 conversation_id 的哈希 )。
func (s *DefaultMessageService) nextSnowflakeID(conversationID string) (uint64, error) {
	shardVal, err := sharding.NewStringSharder(conversationID).ShardVal()
	if err != nil {
		return 0, err
	}
	return s.idgen.NextID(shardVal)
}

// emitIndexEvent 投递消息索引事件 ( 消息落库成功后调用 )。
//
// ES 只是异步查询加速层: 事件丢失可接受 ( reindex 工具可全量重建 )，
// 事件重复可接受 ( ES 以 message_id 为文档 _id 幂等 )——
// 因此投递失败仅记日志，绝不影响消息主链路，也不进入落库事务。
func (s *DefaultMessageService) emitIndexEvent(
	ctx context.Context,
	businessID uint64,
	messageID, conversationID string,
	senderID uint64,
	body []byte,
	contentType string,
	sentAt int64,
) {
	event := domain.MessageIndexEvent{
		MessageID: messageID,

		BusinessID:     businessID,
		ConversationID: conversationID,
		SenderID:       senderID,

		Body:        body,
		ContentType: contentType,

		SentAt: sentAt,
	}

	val, err := json.Marshal(event)
	if err != nil {
		s.logger.Error(
			"[message-service] failed to marshal index event",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		return
	}

	// 分区键 conversation_id: 同会话事件有序 ( 利于 bulk 局部性 )。
	key := []byte(conversationID)
	if err := s.indexProducer.Produce(ctx, key, val); err != nil {
		s.logger.Warn(
			"[message-service] failed to produce index event ( reindex 可补 )",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
	}
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

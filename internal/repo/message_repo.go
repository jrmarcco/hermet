package repo

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// MessageRepo 消息存储仓库。
//
// 落库模型 ( 契约见 synp-api api/README.md 的同步模型 ):
// 单聊一条消息 = 1 次正本写 ( 会话分片 ) + 双方各 1 次收件箱写 ( 用户分片 )。
// 正本与收件箱可能落在不同分片，无法用单库事务覆盖——
// 各分片使用本地事务 + 全链路幂等 ( message_id 唯一键 + ON CONFLICT DO NOTHING )，
// 消费重投时安全 ( seq 允许空洞，只需单调 )。
type MessageRepo interface {
	// StoreSingleChat 落库单聊消息并返回双方收件箱的最新 seq。
	StoreSingleChat(ctx context.Context, msg domain.SingleChatMsg) (domain.SingleChatStoreResult, error)

	// ListInboxEntries 返回 ( cursor, latest_seq ] 内按 seq 升序的至多 limit 条收件箱条目
	// ( limit+1 探测 has_more )，同时返回收件箱当前最大 seq。
	ListInboxEntries(ctx context.Context, businessID, userID uint64, cursor int64, limit int) (entries []domain.InboxEntry, latestSeq int64, hasMore bool, err error)
}

var _ MessageRepo = (*TxMessageRepo)(nil)

type TxMessageRepo struct {
	messageDao dao.MessageDao
	inboxDao   dao.InboxDao
	channelDao dao.ChannelDao

	dbs *xsync.Map[string, *gorm.DB]

	// msgShardHelper 消息正本分片 ( 按 conversation_id )。
	msgShardHelper *sharding.ShardHelper
	// inboxShardHelper 收件箱分片 ( 按 user_id )。
	inboxShardHelper *sharding.ShardHelper
	// channelShardHelper 会话分片 ( 按 user_id，与收件箱同布局 )。
	channelShardHelper *sharding.ShardHelper
}

func NewTxMessageRepo(
	messageDao dao.MessageDao,
	inboxDao dao.InboxDao,
	channelDao dao.ChannelDao,

	dbs *xsync.Map[string, *gorm.DB],
	msgShardHelper *sharding.ShardHelper,
	inboxShardHelper *sharding.ShardHelper,
	channelShardHelper *sharding.ShardHelper,
) *TxMessageRepo {
	return &TxMessageRepo{
		messageDao: messageDao,
		inboxDao:   inboxDao,
		channelDao: channelDao,

		dbs: dbs,

		msgShardHelper:     msgShardHelper,
		inboxShardHelper:   inboxShardHelper,
		channelShardHelper: channelShardHelper,
	}
}

func (r *TxMessageRepo) StoreSingleChat(ctx context.Context, msg domain.SingleChatMsg) (domain.SingleChatStoreResult, error) {
	// 1. 消息正本 ( 会话分片，幂等 )。
	if err := r.saveMessage(ctx, msg); err != nil {
		return domain.SingleChatStoreResult{}, err
	}

	// 2. 双方收件箱写扩散 ( 各自用户分片内本地事务 + 会话快照更新 )。
	senderSeq, err := r.fanoutInbox(ctx, msg, msg.SenderID, false)
	if err != nil {
		return domain.SingleChatStoreResult{}, fmt.Errorf("failed to fanout sender inbox: %w", err)
	}

	receiverSeq, err := r.fanoutInbox(ctx, msg, msg.ReceiverID, true)
	if err != nil {
		return domain.SingleChatStoreResult{}, fmt.Errorf("failed to fanout receiver inbox: %w", err)
	}

	return domain.SingleChatStoreResult{
		SenderSeq:   senderSeq,
		ReceiverSeq: receiverSeq,
	}, nil
}

func (r *TxMessageRepo) saveMessage(ctx context.Context, msg domain.SingleChatMsg) error {
	dst, err := r.msgShardHelper.DstFromSharder(sharding.NewStringSharder(msg.ConversationID))
	if err != nil {
		return fmt.Errorf("failed to get shard destination from conversation id [ %s ]: %w", msg.ConversationID, err)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	entity := dao.Message{
		ID: msg.SnowflakeID,

		MessageID:      msg.MessageID,
		BusinessID:     msg.BusinessID,
		ConversationID: msg.ConversationID,

		SenderID: msg.SenderID,

		Body:        msg.Body,
		ContentType: msg.ContentType,

		SentAt:    msg.SentAt,
		CreatedAt: msg.SentAt,
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.messageDao.SaveInTx(ctx, tx, dst.TB, entity)
	})
}

// fanoutInbox 在目标用户分片内完成: seq 分配 + 收件箱写入 + 会话快照更新。
func (r *TxMessageRepo) fanoutInbox(ctx context.Context, msg domain.SingleChatMsg, receiver uint64, incrementUnread bool) (int64, error) {
	dst, err := r.inboxShardHelper.DstFromSharder(sharding.NewSingleIDSharder(receiver))
	if err != nil {
		return 0, fmt.Errorf("failed to get shard destination from user id [ %d ]: %w", receiver, err)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return 0, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	// 会话分片与收件箱分片同布局 ( 同 db_shard_count/tb_shard_count，同 user_id 路由 )。
	channelDst, err := r.channelShardHelper.DstFromSharder(sharding.NewSingleIDSharder(receiver))
	if err != nil {
		return 0, fmt.Errorf("failed to get channel shard destination from user id [ %d ]: %w", receiver, err)
	}

	var seq int64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		seq, txErr = r.inboxDao.NextSeqInTx(ctx, tx, dst.TB, msg.BusinessID, receiver, 1)
		if txErr != nil {
			return txErr
		}

		entry := dao.InboxEntry{
			BusinessID: msg.BusinessID,
			UserID:     receiver,
			Seq:        seq,

			Kind: dao.InboxEntryKindChat,

			MessageID: msg.MessageID,

			ConversationID: msg.ConversationID,

			SenderID:    msg.SenderID,
			Body:        msg.Body,
			ContentType: msg.ContentType,
			SentAt:      msg.SentAt,

			CreatedAt: msg.SentAt,
		}
		if txErr = r.inboxDao.CreateEntryInTx(ctx, tx, dst.TB, entry); txErr != nil {
			return txErr
		}

		// 会话快照 ( 会话行由 CDC 链路异步创建，行不存在时静默跳过 )。
		target := msg.ReceiverID
		if !incrementUnread {
			target = msg.SenderID
		}
		snapshot := dao.ChannelLastMessage{
			TargetID:    target,
			ChannelType: string(domain.ChannelTypeSingle),

			LastMessageID:       msg.SnowflakeID,
			LastMessageType:     msg.ContentType,
			LastMessageContent:  msg.ContentPreview,
			LastMessageSenderID: msg.SenderID,
			LastMessageTime:     msg.SentAt,

			IncrementUnread: incrementUnread,
		}
		return r.channelDao.UpdateLastMessageInTx(ctx, tx, channelDst.TB, receiver, snapshot)
	})
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (r *TxMessageRepo) ListInboxEntries(
	ctx context.Context,
	businessID, userID uint64,
	cursor int64,
	limit int,
) ([]domain.InboxEntry, int64, bool, error) {
	latestSeq, err := r.inboxDao.LatestSeq(ctx, businessID, userID)
	if err != nil {
		return nil, 0, false, err
	}

	// limit+1 探测是否还有剩余增量。
	entries, err := r.inboxDao.ListEntries(ctx, businessID, userID, cursor, limit+1)
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	domains := make([]domain.InboxEntry, 0, len(entries))
	for _, e := range entries {
		domains = append(domains, toDomainInboxEntry(e))
	}
	return domains, latestSeq, hasMore, nil
}

func toDomainInboxEntry(e dao.InboxEntry) domain.InboxEntry {
	return domain.InboxEntry{
		Seq: e.Seq,

		Kind: domain.InboxEntryKind(e.Kind),

		MessageID: e.MessageID,

		ConversationID:        e.ConversationID,
		ConversationLatestSeq: e.ConversationLatestSeq,

		SenderID:    e.SenderID,
		Body:        e.Body,
		ContentType: e.ContentType,
		SentAt:      e.SentAt,
	}
}

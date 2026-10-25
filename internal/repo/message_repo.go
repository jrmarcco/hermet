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
// 群聊一条消息 = 1 次正本写 ( 会话分片，conversation_seq 同事务分配 ) + N 次收件箱扇出 ( 用户分片 )。
// 正本与收件箱可能落在不同分片，无法用单库事务覆盖——
// 各分片使用本地事务 + 全链路幂等 ( message_id 唯一键 + ON CONFLICT DO NOTHING )，
// 消费重投时安全 ( seq 允许空洞，只需单调 )。
type MessageRepo interface {
	// StoreSingleChat 落库单聊消息并返回双方收件箱的最新 seq。
	StoreSingleChat(ctx context.Context, msg domain.SingleChatMsg) (domain.SingleChatStoreResult, error)

	// StoreGroupChat 落库群聊消息: 会话分片事务内分配 conversation_seq + 写正本，
	// 再按用户分片分组、每 batchSize 成员一个本地事务完成收件箱扇出与会话快照更新。
	// 返回各成员收件箱的最新 seq ( 用于投递通知 )。
	StoreGroupChat(ctx context.Context, msg domain.GroupChatMsg, batchSize int) (domain.GroupChatStoreResult, error)

	// ListInboxEntries 返回 ( cursor, latest_seq ] 内按 seq 升序的至多 limit 条收件箱条目
	// ( limit+1 探测 has_more )，同时返回收件箱当前最大 seq。
	ListInboxEntries(ctx context.Context, businessID, userID uint64, cursor int64, limit int) (entries []domain.InboxEntry, latestSeq int64, hasMore bool, err error)

	// ListConversationTimeline 返回会话 timeline 内 ( cursor, latest_seq ] 按序的至多 limit 条消息
	// ( limit+1 探测 has_more )，同时返回会话当前最大 conversation_seq。
	ListConversationTimeline(ctx context.Context, businessID uint64, conversationID string, cursor int64, limit int) (messages []domain.TimelineMessage, latestSeq int64, hasMore bool, err error)

	// AppendSystemInboxEntry 在指定用户收件箱追加一条系统条目 ( 不写正本、不分配会话 seq )，
	// 返回分配的收件箱 seq。按 ( business_id, user_id, message_id ) 幂等。
	AppendSystemInboxEntry(ctx context.Context, entry domain.InboxEntry) (seq int64, err error)
}

// defaultFanoutBatchSize 群扇出每批成员数兜底值 ( 服务层配置缺省时 )。
const defaultFanoutBatchSize = 500

var _ MessageRepo = (*TxMessageRepo)(nil)

type TxMessageRepo struct {
	messageDao         dao.MessageDao
	conversationSeqDao dao.ConversationSeqDao
	inboxDao           dao.InboxDao
	channelDao         dao.ChannelDao

	dbs *xsync.Map[string, *gorm.DB]

	// msgShardHelper 消息正本分片 ( 按 conversation_id )。
	msgShardHelper *sharding.ShardHelper
	// convSeqShardHelper 会话 seq 分配分片 ( 按 conversation_id，与正本同布局 )。
	convSeqShardHelper *sharding.ShardHelper
	// inboxShardHelper 收件箱分片 ( 按 user_id )。
	inboxShardHelper *sharding.ShardHelper
	// channelShardHelper 会话分片 ( 按 user_id，与收件箱同布局 )。
	channelShardHelper *sharding.ShardHelper
}

func NewTxMessageRepo(
	messageDao dao.MessageDao,
	conversationSeqDao dao.ConversationSeqDao,
	inboxDao dao.InboxDao,
	channelDao dao.ChannelDao,

	dbs *xsync.Map[string, *gorm.DB],
	msgShardHelper *sharding.ShardHelper,
	convSeqShardHelper *sharding.ShardHelper,
	inboxShardHelper *sharding.ShardHelper,
	channelShardHelper *sharding.ShardHelper,
) *TxMessageRepo {
	return &TxMessageRepo{
		messageDao:         messageDao,
		conversationSeqDao: conversationSeqDao,
		inboxDao:           inboxDao,
		channelDao:         channelDao,

		dbs: dbs,

		msgShardHelper:     msgShardHelper,
		convSeqShardHelper: convSeqShardHelper,
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

func (r *TxMessageRepo) StoreGroupChat(ctx context.Context, msg domain.GroupChatMsg, batchSize int) (domain.GroupChatStoreResult, error) {
	if batchSize <= 0 {
		batchSize = defaultFanoutBatchSize
	}

	// 1. 会话分片事务: 分配 conversation_seq + 写正本 ( 原子 )。
	conversationSeq, err := r.saveGroupMessage(ctx, msg)
	if err != nil {
		return domain.GroupChatStoreResult{}, err
	}

	// 2. 收件箱扇出 ( 按用户分片分组 + 分批本地事务 )。
	memberSeqs, err := r.fanoutGroupInbox(ctx, msg, conversationSeq, batchSize)
	if err != nil {
		return domain.GroupChatStoreResult{}, err
	}

	return domain.GroupChatStoreResult{MemberSeqs: memberSeqs}, nil
}

// saveGroupMessage 在会话分片事务内分配 conversation_seq 并写入消息正本。
func (r *TxMessageRepo) saveGroupMessage(ctx context.Context, msg domain.GroupChatMsg) (int64, error) {
	msgDst, err := r.msgShardHelper.DstFromSharder(sharding.NewStringSharder(msg.ConversationID))
	if err != nil {
		return 0, fmt.Errorf("failed to get shard destination from conversation id [ %s ]: %w", msg.ConversationID, err)
	}

	// conversation_seq 与正本同布局路由，落在同一分片库，可共用一个事务。
	seqDst, err := r.convSeqShardHelper.DstFromSharder(sharding.NewStringSharder(msg.ConversationID))
	if err != nil {
		return 0, fmt.Errorf("failed to get seq shard destination from conversation id [ %s ]: %w", msg.ConversationID, err)
	}

	db, ok := r.dbs.Load(msgDst.DB)
	if !ok {
		return 0, fmt.Errorf("failed to load database [ %s ]", msgDst.DB)
	}

	var conversationSeq int64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		conversationSeq, txErr = r.conversationSeqDao.NextSeqInTx(ctx, tx, seqDst.TB, msg.BusinessID, msg.ConversationID, 1)
		if txErr != nil {
			return txErr
		}

		entity := dao.Message{
			ID: msg.SnowflakeID,

			MessageID:      msg.MessageID,
			BusinessID:     msg.BusinessID,
			ConversationID: msg.ConversationID,

			ConversationSeq: conversationSeq,
			SenderID:        msg.SenderID,

			Body:        msg.Body,
			ContentType: msg.ContentType,

			SentAt:    msg.SentAt,
			CreatedAt: msg.SentAt,
		}
		return r.messageDao.SaveInTx(ctx, tx, msgDst.TB, entity)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to save group message: %w", err)
	}

	return conversationSeq, nil
}

// fanoutShardGroup 同一分片库内待扇出的成员批次。
type fanoutShardGroup struct {
	db *gorm.DB

	inboxTB   string
	channelTB string

	userIDs []uint64
}

// fanoutGroupInbox 按用户分片将成员分组后，分批在本地事务内完成:
// 批量 seq 分配 + 批量收件箱写入 ( 内联/指针 ) + 批量会话快照更新。
func (r *TxMessageRepo) fanoutGroupInbox(
	ctx context.Context,
	msg domain.GroupChatMsg,
	conversationSeq int64,
	batchSize int,
) (map[uint64]int64, error) {
	// 1. 按用户分片库分组 ( 收件箱与会话同布局，channelTB 由同路由推导 )。
	groups := make(map[string]*fanoutShardGroup)
	for _, uid := range msg.MemberIDs {
		dst, err := r.inboxShardHelper.DstFromSharder(sharding.NewSingleIDSharder(uid))
		if err != nil {
			return nil, fmt.Errorf("failed to get shard destination from user id [ %d ]: %w", uid, err)
		}

		channelDst, err := r.channelShardHelper.DstFromSharder(sharding.NewSingleIDSharder(uid))
		if err != nil {
			return nil, fmt.Errorf("failed to get channel shard destination from user id [ %d ]: %w", uid, err)
		}

		group, ok := groups[dst.DB]
		if !ok {
			db, dbOk := r.dbs.Load(dst.DB)
			if !dbOk {
				return nil, fmt.Errorf("failed to load database [ %s ]", dst.DB)
			}
			group = &fanoutShardGroup{
				db:        db,
				inboxTB:   dst.TB,
				channelTB: channelDst.TB,
			}
			groups[dst.DB] = group
		}
		group.userIDs = append(group.userIDs, uid)
	}

	entryKind := dao.InboxEntryKindPointer
	if msg.Inline {
		entryKind = dao.InboxEntryKindChat
	}

	snapshot := dao.ChannelLastMessage{
		TargetID:    msg.GroupID,
		ChannelType: string(domain.ChannelTypeGroup),

		LastMessageID:       msg.SnowflakeID,
		LastMessageType:     msg.ContentType,
		LastMessageContent:  msg.ContentPreview,
		LastMessageSenderID: msg.SenderID,
		LastMessageTime:     msg.SentAt,
	}

	memberSeqs := make(map[uint64]int64, len(msg.MemberIDs))

	// 2. 每组内按 batchSize 分批，每批一个本地事务。
	for _, group := range groups {
		for start := 0; start < len(group.userIDs); start += batchSize {
			end := start + batchSize
			if end > len(group.userIDs) {
				end = len(group.userIDs)
			}
			batch := group.userIDs[start:end]

			allocations, err := r.fanoutGroupBatch(ctx, group, batch, msg, conversationSeq, entryKind, snapshot)
			if err != nil {
				return nil, err
			}
			for _, alloc := range allocations {
				memberSeqs[alloc.UserID] = alloc.Seq
			}
		}
	}

	return memberSeqs, nil
}

// fanoutGroupBatch 在单个分片事务内完成一批成员的扇出:
// 批量 seq 分配 + 批量收件箱写入 + 批量会话快照更新 ( 发送者不递增未读数 )。
func (r *TxMessageRepo) fanoutGroupBatch(
	ctx context.Context,
	group *fanoutShardGroup,
	batch []uint64,
	msg domain.GroupChatMsg,
	conversationSeq int64,
	entryKind dao.InboxEntryKind,
	snapshot dao.ChannelLastMessage,
) ([]dao.InboxSeqAllocation, error) {
	var allocations []dao.InboxSeqAllocation

	err := group.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		allocations, txErr = r.inboxDao.NextSeqsBatchInTx(ctx, tx, group.inboxTB, msg.BusinessID, batch)
		if txErr != nil {
			return txErr
		}

		// 批量分配返回顺序与 batch 无关，seq 由分配结果带回。
		seqByUser := make(map[uint64]int64, len(allocations))
		for _, alloc := range allocations {
			seqByUser[alloc.UserID] = alloc.Seq
		}

		entries := make([]dao.InboxEntry, 0, len(batch))
		for _, uid := range batch {
			entry := dao.InboxEntry{
				BusinessID: msg.BusinessID,
				UserID:     uid,
				Seq:        seqByUser[uid],

				Kind: entryKind,

				MessageID: msg.MessageID,

				ConversationID:        msg.ConversationID,
				ConversationLatestSeq: conversationSeq,

				SenderID: msg.SenderID,
				SentAt:   msg.SentAt,

				CreatedAt: msg.SentAt,
			}
			// 写扩散条目内联正本；读扩散指针只携带会话最新 seq。
			if msg.Inline {
				entry.Body = msg.Body
				entry.ContentType = msg.ContentType
			}
			entries = append(entries, entry)
		}
		if txErr = r.inboxDao.CreateEntriesInTx(ctx, tx, group.inboxTB, entries); txErr != nil {
			return txErr
		}

		// 会话快照: 接收者未读 +1，发送者不递增。
		receivers := make([]uint64, 0, len(batch))
		for _, uid := range batch {
			if uid != msg.SenderID {
				receivers = append(receivers, uid)
			}
		}
		if txErr = r.channelDao.UpdateLastMessageForUsersInTx(ctx, tx, group.channelTB, receivers, snapshot, true); txErr != nil {
			return txErr
		}
		return r.channelDao.UpdateLastMessageForUsersInTx(ctx, tx, group.channelTB, []uint64{msg.SenderID}, snapshot, false)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fanout group inbox batch: %w", err)
	}

	return allocations, nil
}

// AppendSystemInboxEntry 在用户分片事务内为系统条目分配 seq 并写入收件箱
// ( 业务级失败回执等场景；不触碰正本与 conversation_seq )。
func (r *TxMessageRepo) AppendSystemInboxEntry(ctx context.Context, entry domain.InboxEntry) (int64, error) {
	dst, err := r.inboxShardHelper.DstFromSharder(sharding.NewSingleIDSharder(entry.UserID))
	if err != nil {
		return 0, fmt.Errorf("failed to get shard destination from user id [ %d ]: %w", entry.UserID, err)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return 0, fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	var seq int64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		seq, txErr = r.inboxDao.NextSeqInTx(ctx, tx, dst.TB, entry.BusinessID, entry.UserID, 1)
		if txErr != nil {
			return txErr
		}

		entity := dao.InboxEntry{
			BusinessID: entry.BusinessID,
			UserID:     entry.UserID,
			Seq:        seq,

			Kind: dao.InboxEntryKind(entry.Kind),

			MessageID: entry.MessageID,

			ConversationID: entry.ConversationID,

			SenderID:    entry.SenderID,
			Body:        entry.Body,
			ContentType: entry.ContentType,
			SentAt:      entry.SentAt,

			CreatedAt: entry.SentAt,
		}
		return r.inboxDao.CreateEntryInTx(ctx, tx, dst.TB, entity)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to append system inbox entry: %w", err)
	}
	return seq, nil
}

func (r *TxMessageRepo) ListConversationTimeline(
	ctx context.Context,
	businessID uint64,
	conversationID string,
	cursor int64,
	limit int,
) ([]domain.TimelineMessage, int64, bool, error) {
	latestSeq, err := r.conversationSeqDao.LatestSeq(ctx, businessID, conversationID)
	if err != nil {
		return nil, 0, false, err
	}

	// limit+1 探测是否还有剩余增量。
	messages, err := r.messageDao.ListByConversation(ctx, businessID, conversationID, cursor, limit+1)
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	domains := make([]domain.TimelineMessage, 0, len(messages))
	for _, m := range messages {
		domains = append(domains, domain.TimelineMessage{
			ConversationSeq: m.ConversationSeq,

			MessageID:      m.MessageID,
			ConversationID: m.ConversationID,
			SenderID:       m.SenderID,

			Body:        m.Body,
			ContentType: m.ContentType,

			SentAt: m.SentAt,
		})
	}
	return domains, latestSeq, hasMore, nil
}

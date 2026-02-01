package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

type ChannelRepo interface {
	// Create 创建用户会话。
	Create(ctx context.Context, channel domain.Channel) error

	ListByUserID(ctx context.Context, userID uint64) ([]domain.Channel, error)
}

var _ ChannelRepo = (*DefaultChannelRepo)(nil)

type DefaultChannelRepo struct {
	channelDao     dao.ChannelDao
	channelRidxDao dao.ChannelRidxDao

	dbs              *xsync.Map[string, *gorm.DB]
	cShardHelper     *sharding.ShardHelper
	cRidxShardHelper *sharding.ShardHelper
}

func NewDefaultChannelRepo(
	channelDao dao.ChannelDao,
	channelRidxDao dao.ChannelRidxDao,
	dbs *xsync.Map[string, *gorm.DB],
	cShardHelper *sharding.ShardHelper,
	cRidxShardHelper *sharding.ShardHelper,
) *DefaultChannelRepo {
	return &DefaultChannelRepo{
		channelDao:       channelDao,
		channelRidxDao:   channelRidxDao,
		dbs:              dbs,
		cShardHelper:     cShardHelper,
		cRidxShardHelper: cRidxShardHelper,
	}
}

func (r *DefaultChannelRepo) Create(ctx context.Context, channel domain.Channel) error {
	// 1. 准备 Sharder ( 以 target_id 为分片键获取 Sharder )。
	sharder := sharding.NewSingleIDSharder(channel.TargetID)

	// 2. 生成 channel ID 并获取分片信息。
	id, dst, err := r.cShardHelper.NextIDAndShard(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from channel target id [ %d ]: %w", channel.TargetID, err)
	}

	// 3. 加载 gorm.DB 连接。
	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	// 4. 获取反向索引表的分片信息。
	criDst, _ := r.cRidxShardHelper.DstFromSharder(sharder)

	now := time.Now().UnixMilli()

	// 5. 开启事务协调 DAO 写入数据。
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 6. 写入 channel 表。
		channelEntity := dao.Channel{
			ID: id,

			UserID:        channel.UserID,
			TargetID:      channel.TargetID,
			TargetInfoVer: channel.TargetInfoVer,

			ChannelType:   string(channel.ChannelType),
			ChannelName:   channel.ChannelName,
			ChannelRemark: channel.ChannelRemark,
			ChannelAvatar: channel.ChannelAvatar,

			OpenedAt: channel.OpenedAt,

			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.channelDao.CreateInTx(ctx, tx, dst.TB, channelEntity); err != nil {
			return fmt.Errorf("failed to create channel in transaction: %w", err)
		}

		// 7. 写入 channel_reverse_index 表。
		channelRidxEntity := dao.ChannelReverseIndex{
			ID: channelEntity.ID,

			UserID:      channelEntity.UserID,
			TargetID:    channelEntity.TargetID,
			ChannelType: channelEntity.ChannelType,

			CreatedAt: now,
		}
		if err := r.channelRidxDao.CreateInTx(ctx, tx, criDst.TB, channelRidxEntity); err != nil {
			return fmt.Errorf("failed to create channel reverse index in transaction: %w", err)
		}
		return nil
	})
}

func (r *DefaultChannelRepo) ListByUserID(ctx context.Context, userID uint64) ([]domain.Channel, error) {
	entities, err := r.channelDao.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.Channel, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultChannelRepo) toDomain(channel dao.Channel) domain.Channel {
	return domain.Channel{
		ID: channel.ID,

		UserID:        channel.UserID,
		TargetID:      channel.TargetID,
		TargetInfoVer: channel.TargetInfoVer,

		ChannelType:   domain.ChannelType(channel.ChannelType),
		ChannelName:   channel.ChannelName,
		ChannelRemark: channel.ChannelRemark,
		ChannelAvatar: channel.ChannelAvatar,

		IsMuted:  channel.IsMuted,
		IsPinned: channel.IsPinned,

		LastMessageID:         channel.LastMessageID,
		LastMessageType:       channel.LastMessageType,
		LastMessageContent:    channel.LastMessageContent,
		LastMessageSenderID:   channel.LastMessageSenderID,
		LastMessageSenderName: channel.LastMessageSenderName,
		LastMessageTime:       channel.LastMessageTime,

		UnreadCount:  channel.UnreadCount,
		MentionCount: channel.MentionCount,

		OpenedAt: channel.OpenedAt,
		ClosedAt: channel.ClosedAt,

		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}

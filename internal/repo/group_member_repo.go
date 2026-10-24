package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

type GroupMemberRepo interface {
	BatchCreate(ctx context.Context, groupID, creatorID uint64, memberIDs []uint64) error
	CreateView(ctx context.Context, gmv domain.GroupMemberView) error

	// ListActiveMemberIDs 返回群内有效成员 ID 列表 ( 含发送者，用于群消息扇出 )。
	ListActiveMemberIDs(ctx context.Context, groupID uint64) ([]uint64, error)
}

var _ GroupMemberRepo = (*TxGroupMemberRepo)(nil)

type TxGroupMemberRepo struct {
	userDao        dao.UserDao
	outboxEventDao dao.OutboxEventDao

	groupMemberDao     dao.GroupMemberDao
	groupMemberViewDao dao.GroupMemberViewDao

	dbs            *xsync.Map[string, *gorm.DB]
	oeShardHelper  *sharding.ShardHelper
	gmvShardHelper *sharding.ShardHelper
}

func NewTxGroupMemberRepo(
	userDao dao.UserDao,
	outboxEventDao dao.OutboxEventDao,
	groupMemberDao dao.GroupMemberDao,
	groupMemberViewDao dao.GroupMemberViewDao,
	dbs *xsync.Map[string, *gorm.DB],
	oeShardHelper, gmvShardHelper *sharding.ShardHelper,
) *TxGroupMemberRepo {
	return &TxGroupMemberRepo{
		userDao:        userDao,
		outboxEventDao: outboxEventDao,

		groupMemberDao:     groupMemberDao,
		groupMemberViewDao: groupMemberViewDao,

		dbs:            dbs,
		oeShardHelper:  oeShardHelper,
		gmvShardHelper: gmvShardHelper,
	}
}

func (r *TxGroupMemberRepo) BatchCreate(ctx context.Context, groupID, creatorID uint64, memberIDs []uint64) error {
	return r.groupMemberDao.BatchCreate(ctx, groupID, creatorID, memberIDs)
}

func (r *TxGroupMemberRepo) ListActiveMemberIDs(ctx context.Context, groupID uint64) ([]uint64, error) {
	return r.groupMemberDao.ListActiveMemberIDs(ctx, groupID)
}

func (r *TxGroupMemberRepo) CreateView(ctx context.Context, gmv domain.GroupMemberView) error {
	// 1. 查询联系人信息 ( 用于群成员信息 )。
	user, err := r.userDao.FindByID(ctx, gmv.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	// 2. 获取 group_id 对应的分片信息并加载 gorm.DB 连接。
	dst, err := r.gmvShardHelper.DstFromID(gmv.ID)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", gmv.ID)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	// 3. 获取 outbox_event 表的分片信息。
	oeDst, _ := r.oeShardHelper.DstFromID(gmv.ID)

	now := time.Now().UnixMilli()

	// 4. 开启事务协调 DAO 写入数据。
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 5. 写入 group_member_view 表。
		gmvEntity := dao.GroupMemberView{
			ID: gmv.ID,

			GroupID: gmv.GroupID,
			UserID:  gmv.UserID,

			MemberRole: string(gmv.MemberRole),

			MemberNickname: user.Nickname,
			MemberAvatar:   user.Avatar,
			MemberGender:   user.Gender,
			MemberRemark:   user.Nickname,
			MemberInfoVer:  user.InfoVer,

			JoinedAt:  gmv.JoinedAt,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.groupMemberViewDao.CreateInTx(ctx, tx, dst.TB, gmvEntity); err != nil {
			return fmt.Errorf("failed to create group member view in transaction: %w", err)
		}

		// 6. 写入 outbox_event 表 ( 分流 CDC 事件用以创建 user_group 记录 )。
		payload, err := json.Marshal(domain.GroupMemberOutboxEvent{
			GroupID:  gmv.GroupID,
			UserID:   gmv.UserID,
			JoinedAt: gmv.JoinedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal outbox event payload: %w", err)
		}

		outboxEventEntity := dao.OutboxEvent{
			ID: gmv.ID,

			AggregateID:   gmv.GroupID,
			AggregateType: string(domain.OutboxAggregateTypeGm),

			EventType: string(domain.OutboxEventTypeGmCreate),
			Payload:   payload,
			Metadata:  []byte("{}"),

			CreatedAt: now,
		}

		if err := r.outboxEventDao.CreateInTx(ctx, tx, oeDst.TB, outboxEventEntity); err != nil {
			return fmt.Errorf("failed to create outbox event in transaction: %w", err)
		}
		return nil
	})
}

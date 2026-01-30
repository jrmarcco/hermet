package repo

import (
	"context"
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

type GroupRepo interface {
	Create(ctx context.Context, group domain.ChatGroup) error
	CreateMemberView(ctx context.Context, gmv domain.GroupMemberView) error
	AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error

	UpdateStatus(ctx context.Context, id uint64, status string) error

	FindByID(ctx context.Context, id uint64) (domain.ChatGroup, error)
}

var _ GroupRepo = (*TxGroupRepo)(nil)

type TxGroupRepo struct {
	userDao              dao.UserDao
	groupDao             dao.GroupDao
	groupMemberDao       dao.GroupMemberDao
	groupMemberViewDao   dao.GroupMemberViewDao
	groupMemberOutboxDao dao.GroupMemberOutboxDao

	dbs            *xsync.Map[string, *gorm.DB]
	gShardHelper   *sharding.ShardHelper
	gmShardHelper  *sharding.ShardHelper
	gmvShardHelper *sharding.ShardHelper
	gmoShardHelper *sharding.ShardHelper
}

func NewTxGroupRepo(
	userDao dao.UserDao,
	groupDao dao.GroupDao,
	groupMemberDao dao.GroupMemberDao,
	groupMemberViewDao dao.GroupMemberViewDao,
	groupMemberOutboxDao dao.GroupMemberOutboxDao,
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper, gmShardHelper, gmvShardHelper, gmoShardHelper *sharding.ShardHelper,
) *TxGroupRepo {
	return &TxGroupRepo{
		userDao:              userDao,
		groupDao:             groupDao,
		groupMemberDao:       groupMemberDao,
		groupMemberViewDao:   groupMemberViewDao,
		groupMemberOutboxDao: groupMemberOutboxDao,

		dbs:            dbs,
		gShardHelper:   shardHelper,
		gmShardHelper:  gmShardHelper,
		gmvShardHelper: gmvShardHelper,
		gmoShardHelper: gmoShardHelper,
	}
}

func (r *TxGroupRepo) Create(ctx context.Context, group domain.ChatGroup) error {
	return r.groupDao.Create(ctx, dao.ChatGroup{
		ID: group.ID,

		GroupName:        group.GroupName,
		GroupAvatar:      group.GroupAvatar,
		GroupStatus:      string(group.GroupStatus),
		GroupMemberCount: group.GroupMemberCount,

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	})
}

func (r *TxGroupRepo) CreateMemberView(ctx context.Context, gmv domain.GroupMemberView) error {
	// 1. 查询联系人信息 ( 用于群成员信息 )。
	user, err := r.userDao.FindByID(ctx, gmv.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	// 2. 获取 group_id 对应的分片信息并加载 gorm.DB 连接。
	dst, err := r.gmShardHelper.DstFromID(gmv.ID)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", gmv.ID)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	// 3. 获取 group_member_outbox 表的分片信息。
	gmoDst, _ := r.gmoShardHelper.DstFromID(gmv.ID)

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

		// 6. 写入 group_member_outbox 表 ( 分流 CDC 事件用以创建 user_group 记录 )。
		gmoEntity := dao.GroupMemberOutboxEvent{
			ID: gmv.ID,

			GroupID: gmv.GroupID,
			UserID:  gmv.UserID,
			// 操作设置为创建群成员。
			OpType: string(domain.GroupMemberOpTypeCreate),

			JoinedAt:  gmv.JoinedAt,
			CreatedAt: now,
		}
		if err := r.groupMemberOutboxDao.CreateInTx(ctx, tx, gmoDst.TB, gmoEntity); err != nil {
			return fmt.Errorf("failed to create group member outbox in transaction: %w", err)
		}
		return nil
	})
}

func (r *TxGroupRepo) AddMembers(ctx context.Context, groupID uint64, memberIDs []uint64) error {
	return r.groupMemberDao.BatchCreate(ctx, groupID, memberIDs)
}

func (r *TxGroupRepo) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.groupDao.UpdateStatus(ctx, id, status)
}

func (r *TxGroupRepo) FindByID(ctx context.Context, id uint64) (domain.ChatGroup, error) {
	group, err := r.groupDao.FindByID(ctx, id)
	if err != nil {
		return domain.ChatGroup{}, err
	}
	return domain.ChatGroup{
		ID: group.ID,

		GroupName:        group.GroupName,
		GroupAvatar:      group.GroupAvatar,
		GroupStatus:      domain.ChatGroupStatus(group.GroupStatus),
		GroupMemberCount: group.GroupMemberCount,

		InfoVer: group.InfoVer,

		LastMessageAt: group.LastMessageAt,

		CreatorID: group.CreatorID,

		DeletedAt: group.DeletedAt,

		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}, nil
}

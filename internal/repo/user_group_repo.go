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

type UserGroupRepo interface {
	Create(ctx context.Context, ug domain.UserGroup) error
}

type TxUserGroupRepo struct {
	groupDao         dao.GroupDao
	userGroupDao     dao.UserGroupDao
	userGroupViewDao dao.UserGroupViewDao
	userGroupRidxDao dao.UserGroupRidxDao

	dbs             *xsync.Map[string, *gorm.DB]
	ugShardHelper   *sharding.ShardHelper
	ugvShardHelper  *sharding.ShardHelper
	ugriShardHelper *sharding.ShardHelper
}

func NewTxUserGroupRepo(
	groupDao dao.GroupDao,
	userGroupDao dao.UserGroupDao,
	userGroupViewDao dao.UserGroupViewDao,
	userGroupRidxDao dao.UserGroupRidxDao,
	dbs *xsync.Map[string, *gorm.DB],
	ugShardHelper, ugvShardHelper, ugriShardHelper *sharding.ShardHelper,
) *TxUserGroupRepo {
	return &TxUserGroupRepo{
		groupDao: groupDao,

		userGroupDao:     userGroupDao,
		userGroupViewDao: userGroupViewDao,
		userGroupRidxDao: userGroupRidxDao,

		dbs:             dbs,
		ugShardHelper:   ugShardHelper,
		ugvShardHelper:  ugvShardHelper,
		ugriShardHelper: ugriShardHelper,
	}
}

func (r *TxUserGroupRepo) Create(ctx context.Context, ug domain.UserGroup) error {
	// 1. 查询群组信息 。
	group, err := r.groupDao.FindByID(ctx, ug.GroupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	// 2. 准备 Sharder ( 以 user_id 为分片键获取 Sharder )。
	sharder := sharding.NewSingleIDSharder(ug.UserID)

	// 3. 生成 user_group 表的 ID 并获取分片信息。
	id, dst, err := r.ugShardHelper.NextIDAndShard(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", ug.UserID)
	}

	// 4. 获取试图表以及反向索引表的分片信息。
	ugvDst, _ := r.ugvShardHelper.DstFromSharder(sharder)
	ugriDst, _ := r.ugriShardHelper.DstFromSharder(sharder)

	// 5. 加载 gorm.DB 连接。
	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 6. 开启事务协调 DAO 写入数据。
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 7. 写入 user_group 表。
		ugEntity := dao.UserGroup{
			ID: id,

			UserID:  ug.UserID,
			GroupID: ug.GroupID,

			RemarkName: ug.RemarkName,

			JoinedAt:  ug.JoinedAt,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.userGroupDao.CreateInTx(ctx, tx, dst.TB, ugEntity); err != nil {
			return fmt.Errorf("failed to create user group in transaction: %w", err)
		}

		// 8. 写入 user_group_view 表。
		ugvEntity := dao.UserGroupView{
			ID: id,

			UserID:  ug.UserID,
			GroupID: ug.GroupID,

			GroupName:    group.GroupName,
			GroupAvatar:  group.GroupAvatar,
			GroupStatus:  group.GroupStatus,
			GroupInfoVer: group.InfoVer,

			RemarkName: ug.RemarkName,

			JoinedAt:  ug.JoinedAt,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.userGroupViewDao.CreateInTx(ctx, tx, ugvDst.TB, ugvEntity); err != nil {
			return fmt.Errorf("failed to create user group view in transaction: %w", err)
		}

		// 9. 写入 user_group_reverse_index 表。
		ugriEntity := dao.UserGroupReverseIndex{
			ID: id,

			UserID:  ug.UserID,
			GroupID: ug.GroupID,

			CreatedAt: now,
		}
		if err := r.userGroupRidxDao.CreateInTx(ctx, tx, ugriDst.TB, ugriEntity); err != nil {
			return fmt.Errorf("failed to create user group reverse index in transaction: %w", err)
		}
		return nil
	})
}

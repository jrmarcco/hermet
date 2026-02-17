package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type ContactRepo interface {
	// Create 创建用户联系人。
	Create(ctx context.Context, contact domain.Contact) error
	// Recreate 重新创建用户联系人 ( 用于删除后重新添加 )。
	Recreate(ctx context.Context, contact domain.Contact) error

	FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.Contact, error)
	ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ContactView, error)
}

var _ ContactRepo = (*TxContactRepo)(nil)

// TxContactRepo 是用户联系人事务操作的仓库。
// 需要开启事务协调多个 DAO 的数据写入。
type TxContactRepo struct {
	userDao        dao.UserDao
	contactDao     dao.ContactDao
	contactViewDao dao.ContactViewDao
	contactRidxDao dao.ContactRidxDao

	dbs            *xsync.Map[string, *gorm.DB]
	cShardHelper   *sharding.ShardHelper
	cvShardHelper  *sharding.ShardHelper
	criShardHelper *sharding.ShardHelper
}

func NewTxContactRepo(
	userDao dao.UserDao,
	contactDao dao.ContactDao,
	contactViewDao dao.ContactViewDao,
	contactRidxDao dao.ContactRidxDao,
	dbs *xsync.Map[string, *gorm.DB],
	shardHelper, cvShardHelper, criShardHelper *sharding.ShardHelper,
) *TxContactRepo {
	return &TxContactRepo{
		dbs: dbs,

		userDao:        userDao,
		contactDao:     contactDao,
		contactViewDao: contactViewDao,
		contactRidxDao: contactRidxDao,

		cShardHelper:   shardHelper,
		cvShardHelper:  cvShardHelper,
		criShardHelper: criShardHelper,
	}
}

// Create 创建用户联系人。
// 这里需要在同一个事务内写入多个数据表：
//   - contact
//   - contact_view
//   - contact_reverse_index
//
// 这 3 张表都是以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作。
func (r *TxContactRepo) Create(ctx context.Context, contact domain.Contact) error {
	// 1. 查询联系人信息 ( 用于冗余联系人信息 )。
	user, err := r.userDao.FindByID(ctx, contact.ContactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	// 2. 准备 Sharder ( 以 user_id 为分片键获取 Sharder )。
	sharder := sharding.NewSingleIDSharder(contact.UserID)

	// 3. 生成联系人 ID 并获取分片信息。
	id, dst, err := r.cShardHelper.NextIDAndShard(sharder)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", contact.UserID)
	}

	// 4. 获取视图表以及反向索引表的分片信息。
	//	注意：
	// 	视图表和反向索引表和联系人表使用同一个 Sharder。
	//	理论上这里不会出现 error，因此忽略 error 返回值。
	cvDst, _ := r.cvShardHelper.DstFromSharder(sharder)
	criDst, _ := r.criShardHelper.DstFromSharder(sharder)

	// 5. 加载 gorm.DB 连接。
	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	// 开启事务协调 DAO 写入数据。
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 6. 写入 contact 表。
		contactEntity := dao.Contact{
			ID: id,

			UserID:    contact.UserID,
			ContactID: contact.ContactID,

			RemarkName: contact.RemarkName,
			Source:     string(contact.Source),

			AddedAt: contact.AddedAt,

			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.contactDao.CreateInTx(ctx, tx, dst.TB, contactEntity); err != nil {
			return fmt.Errorf("failed to create user contact in transaction: %w", err)
		}

		// 7. 写入 contact_view 表。
		viewEntity := dao.ContactView{
			ID: contactEntity.ID,

			UserID:    contact.UserID,
			ContactID: contact.ContactID,
			// 冗余用户信息。
			ContactNickname: user.Nickname,
			ContactAvatar:   user.Avatar,
			ContactGender:   user.Gender,
			ContactMobile:   user.Mobile,
			ContactInfoVer:  user.InfoVer,

			IsStarred: false,
			IsBlocked: false,
			IsMuted:   false,
			IsPinned:  false,

			RemarkName: contactEntity.RemarkName,
			Source:     string(contact.Source),

			AddedAt:   contact.AddedAt,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := r.contactViewDao.CreateInTx(ctx, tx, cvDst.TB, viewEntity); err != nil {
			return fmt.Errorf("failed to create user contact view in transaction: %w", err)
		}

		// 6. 写入 contact_reverse_index 表。
		ridxEntity := dao.ContactReverseIndex{
			ID: contactEntity.ID,

			UserID:  contact.UserID,
			OwnerID: contact.ContactID,

			CreatedAt: now,
		}
		if err := r.contactRidxDao.CreateInTx(ctx, tx, criDst.TB, ridxEntity); err != nil {
			return fmt.Errorf("failed to create user contact reverse index in transaction: %w", err)
		}
		return nil
	})
}

func (r *TxContactRepo) Recreate(ctx context.Context, contact domain.Contact) error {
	// 1. 查询联系人信息 ( 用于冗余联系人信息 )。
	user, err := r.userDao.FindByID(ctx, contact.ContactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	// 2. 获取联系人 ID 对应的分片信息并加载 gorm.DB 连接。
	dst, err := r.cShardHelper.DstFromID(contact.ID)
	if err != nil {
		return fmt.Errorf("failed to get shard destination from id [ %d ]", contact.ID)
	}

	db, ok := r.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	// 3. 获取视图表以及反向索引表的分片信息。
	cvDst, _ := r.cvShardHelper.DstFromID(contact.ID)
	criDst, _ := r.criShardHelper.DstFromID(contact.ID)

	now := time.Now().UnixMilli()

	// 4. 开启事务协调 DAO 写入数据。
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 5. 更新 contact 表。
		contactEntity := dao.Contact{
			ID: contact.ID,
			// 只赋值待更新的字段。
			AddedAt:   contact.AddedAt,
			UpdatedAt: now,
		}
		if err := r.contactDao.RecreateInTx(ctx, tx, dst.TB, contactEntity); err != nil {
			return fmt.Errorf("failed to recreate user contact in transaction: %w", err)
		}

		// 6. 更新 contact_view 表。
		viewEntity := dao.ContactView{
			ID: contactEntity.ID,

			UserID:    contact.UserID,
			ContactID: contact.ContactID,
			// 冗余用户信息。
			ContactNickname: user.Nickname,
			ContactAvatar:   user.Avatar,
			ContactInfoVer:  user.InfoVer,
			ContactGender:   user.Gender,
			ContactMobile:   user.Mobile,

			// 重置状态。
			IsStarred: false,
			IsBlocked: false,
			IsMuted:   false,
			IsPinned:  false,

			RemarkName: contactEntity.RemarkName,
			Source:     string(contact.Source),

			AddedAt:   contact.AddedAt,
			UpdatedAt: now,
		}
		if err := r.contactViewDao.RecreateInTx(ctx, tx, cvDst.TB, viewEntity); err != nil {
			return fmt.Errorf("failed to recreate user contact view in transaction: %w", err)
		}

		// 7. 写入 contact_reverse_index 表。
		//    contact_reverse_index 没有逻辑删除字段，需要重新创建一条记录。
		ridxEntity := dao.ContactReverseIndex{
			ID: contactEntity.ID,

			UserID:  contact.UserID,
			OwnerID: contact.ContactID,

			CreatedAt: now,
		}
		if err := r.contactRidxDao.CreateInTx(ctx, tx, criDst.TB, ridxEntity); err != nil {
			return fmt.Errorf("failed to create user contact reverse index in transaction: %w", err)
		}
		return nil
	})
}

func (r *TxContactRepo) FindByUserIDAndContactID(ctx context.Context, userID, contactID uint64) (domain.Contact, error) {
	entity, err := r.contactDao.FindByUserIDAndContactID(ctx, userID, contactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Contact{}, errs.ErrRecordNotFound
		}
		return domain.Contact{}, err
	}
	return r.toDomain(entity), nil
}

func (r *TxContactRepo) ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ContactView, error) {
	entities, err := r.contactViewDao.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.ContactView, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toViewDomain(entities[i]))
	}
	return domains, nil
}

func (r *TxContactRepo) toDomain(uc dao.Contact) domain.Contact {
	return domain.Contact{
		ID: uc.ID,

		UserID:    uc.UserID,
		ContactID: uc.ContactID,

		RemarkName: uc.RemarkName,
		Source:     domain.ContactSource(uc.Source),

		IsStarred: uc.IsStarred,
		IsBlocked: uc.IsBlocked,
		IsMuted:   uc.IsMuted,
		IsPinned:  uc.IsPinned,

		AddedAt:   uc.AddedAt,
		DeletedAt: uc.DeletedAt,

		CreatedAt: uc.CreatedAt,
		UpdatedAt: uc.UpdatedAt,
	}
}

func (r *TxContactRepo) toViewDomain(ucv dao.ContactView) domain.ContactView {
	return domain.ContactView{
		ID: ucv.ID,

		UserID:    ucv.UserID,
		ContactID: ucv.ContactID,

		ContactNickname: ucv.ContactNickname,
		ContactAvatar:   ucv.ContactAvatar,
		ContactGender:   ucv.ContactGender,
		ContactMobile:   ucv.ContactMobile,
		ContactInfoVer:  ucv.ContactInfoVer,

		RemarkName: ucv.RemarkName,

		IsStarred: ucv.IsStarred,
		IsBlocked: ucv.IsBlocked,
		IsMuted:   ucv.IsMuted,
		IsPinned:  ucv.IsPinned,

		Source: domain.ContactSource(ucv.Source),

		AddedAt: ucv.AddedAt,
	}
}

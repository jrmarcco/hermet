package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"gorm.io/gorm"
)

// ContactApprovalDao 联系人申请审批 dao 接口。
//
//	这是一个特殊的聚合 dao。
//	和 dao 一样持有 dbs ( *xsync.Map[string, *gorm.DB] ) 和 shardHelper ( *sharding.ShardHelper ) 。
//	用来开启事务聚合同一个数据库分片内的多个 dao 操作。
type ContactApprovalDao interface {
	SaveUserRelation(ctx context.Context, userID, contactID uint64, source string) error
}

var _ ContactApprovalDao = (*TxContactApprovalDao)(nil)

type TxContactApprovalDao struct {
	dbs         *xsync.Map[string, *gorm.DB]
	shardHelper *sharding.ShardHelper
}

func NewTxContactApprovalDao(dbs *xsync.Map[string, *gorm.DB], shardHelper *sharding.ShardHelper) *TxContactApprovalDao {
	return &TxContactApprovalDao{
		dbs:         dbs,
		shardHelper: shardHelper,
	}
}

// SaveUserRelation 保存用户关系。
// 联系人申请审批通过后，建立用户之间关联关系。
//
//	A -> B 的关联关系数据以 A 用户的 user_id 为分片键。
//	B -> A 的关联关系数据以 B 用户的 user_id 为分片键。
//	因此这里只创建 A -> B 的关联关系 ( B -> A 的关系需要通过消费 debezium 消息进行建立 )。
//
// 需要写入的表：
// ( 这几个表都以 user_id 为分片键，因此能确保在同一个数据库分片内进行事务操作 )
//   - user_contact
//   - user_contact_view
//   - contact_reverse_index
//   - user_conversation_view
//   - conversation_reverse_index
func (d *TxContactApprovalDao) SaveUserRelation(ctx context.Context, userID, contactID uint64, source string) error {
	// 1. 获取分片并开启事务。
	dst, err := d.shardHelper.DstFromSharder(sharding.NewSingleIDSharder(userID))
	if err != nil {
		return fmt.Errorf("failed to get shard destination from user id [ %d ]", userID)
	}

	db, ok := d.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load database [ %s ]", dst.DB)
	}

	now := time.Now().UnixMilli()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 2. 写入 user_contact 表。
		uc := UserContact{
			UserID:    userID,
			ContactID: contactID,
			Source:    source,
			IsStarred: false,
			IsBlocked: false,
			CreatedAt: now,
			UpdatedAt: now,
		}
		ucID, dst, err := d.shardHelper.NextIDAndShard(sharding.NewSingleIDSharder(userID))
		if err != nil {
			return fmt.Errorf("failed to get next id and shard: %w", err)
		}
		uc.ID = ucID
		err = tx.WithContext(ctx).Table(dst.TB).Model(&UserContact{}).Create(&uc).Error
		if err != nil {
			return fmt.Errorf("failed to save user contact: %w", err)
		}

		// 3. 写入 user_contact_view 表。

		// 4. 写入 contact_reverse_index 表。

		// 5. 写入 user_conversation_view 表。

		// 6. 写入 conversation_reverse_index 表。
		return nil
	})
}

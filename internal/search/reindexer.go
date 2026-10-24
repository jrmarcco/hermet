package search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/jit/xsync"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// reindex 分页参数。
const reindexPageSize = 1000

// Reindexer 从 PG message 表全量扫描重建 ES 索引。
//
// 用途: 索引 mapping 变更迁移 / 索引事件丢失补数 / ES 数据重建。
// 语义: 广播所有 message 分片表，按雪花 id 游标分页，文档 _id = message_id 覆盖写——
// 重复执行幂等，可与在线索引链路并行 ( 覆盖写不冲突 )。
type Reindexer struct {
	dbs      *xsync.Map[string, *gorm.DB]
	msgShard *sharding.ShardHelper

	client *elasticsearch.Client
	index  string

	logger *zap.Logger
}

func NewReindexer(
	dbs *xsync.Map[string, *gorm.DB],
	msgShard *sharding.ShardHelper,
	client *elasticsearch.Client,
	index string,
	logger *zap.Logger,
) *Reindexer {
	return &Reindexer{
		dbs:      dbs,
		msgShard: msgShard,
		client:   client,
		index:    index,
		logger:   logger,
	}
}

// reindexRow 扫描行 ( 雪花 id 仅作分页游标，不写入文档 )。
type reindexRow struct {
	ID uint64 `gorm:"column:id"`

	domain.MessageIndexEvent
}

// Run 执行全量重建，返回索引文档数。
func (r *Reindexer) Run(ctx context.Context) (int64, error) {
	if err := EnsureMessageIndex(ctx, r.client, r.index); err != nil {
		return 0, err
	}

	bulk, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: r.client,
		Index:  r.index,

		NumWorkers:    2,               //nolint:mnd // 重建工具固定双 worker
		FlushBytes:    5 * 1024 * 1024, //nolint:mnd // 5MB 攒批阈值
		FlushInterval: 2 * time.Second, //nolint:mnd // 2s 兜底 flush

		OnError: func(_ context.Context, err error) {
			r.logger.Error("[reindexer] bulk indexer error", zap.Error(err))
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create bulk indexer: %w", err)
	}
	defer func() {
		_ = bulk.Close(context.Background())
	}()

	var total int64

	// 广播 message 全部分片表 ( db_shard_count * tb_shard_count )。
	for _, dst := range r.msgShard.Broadcast() {
		count, tableErr := r.reindexTable(ctx, bulk, dst.DB, dst.TB)
		if tableErr != nil {
			return total, tableErr
		}
		total += count

		r.logger.Info("[reindexer] table done",
			zap.String("table", dst.DB+"."+dst.TB),
			zap.Int64("indexed", count),
		)
	}

	return total, nil
}

// reindexTable 按雪花 id 游标分页扫描单表并写入 bulk。
func (r *Reindexer) reindexTable(ctx context.Context, bulk esutil.BulkIndexer, dbName, tbName string) (int64, error) {
	db, ok := r.dbs.Load(dbName)
	if !ok {
		return 0, fmt.Errorf("failed to load database [ %s ]", dbName)
	}

	var count int64
	var cursor uint64

	for {
		rows := make([]reindexRow, 0, reindexPageSize)
		if err := db.WithContext(ctx).
			Table(tbName).
			Select("id, message_id, business_id, conversation_id, sender_id, body, content_type, sent_at").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(reindexPageSize).
			Find(&rows).Error; err != nil {
			return count, fmt.Errorf("failed to scan table [ %s.%s ]: %w", dbName, tbName, err)
		}

		if len(rows) == 0 {
			return count, nil
		}

		for _, row := range rows {
			doc, err := json.Marshal(row.MessageIndexEvent)
			if err != nil {
				return count, fmt.Errorf("failed to marshal message [ %s ]: %w", row.MessageID, err)
			}

			if err = bulk.Add(ctx, esutil.BulkIndexerItem{
				Action: "index",

				DocumentID: row.MessageID,
				Body:       jsonReader(doc),
			}); err != nil {
				return count, fmt.Errorf("failed to enqueue message [ %s ]: %w", row.MessageID, err)
			}
			count++
		}

		// 游标推进到本页最后一条的雪花 id。
		cursor = rows[len(rows)-1].ID
	}
}

package search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
)

// Indexer 消费消息索引事件并批量写入 ES。
//
// 写入语义:
//   - 文档 _id = message_id，链路 at-least-once 重复消费幂等 ( index 覆盖写 );
//   - esutil.BulkIndexer 内部攒批 ( worker/字节/间隔三阈值 flush );
//   - 单条事件非法 ( 解析失败/字段缺失 ) 跳过不重投 ( ES 可由 reindex 重建 );
//   - bulk 写入失败返回 error，Kafka 按消费语义重投 ( 幂等 )。
type Indexer struct {
	bulk esutil.BulkIndexer

	index string
}

// BulkConfig bulk indexer 攒批参数 ( es 前缀配置段 )。
type BulkConfig struct {
	NumWorkers    int
	FlushBytes    int
	FlushInterval time.Duration
}

func NewIndexer(
	client *elasticsearch.Client,
	index string,
	bulkCfg BulkConfig,
) (*Indexer, error) {
	bulk, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: client,

		Index: index,

		NumWorkers:    bulkCfg.NumWorkers,
		FlushBytes:    bulkCfg.FlushBytes,
		FlushInterval: bulkCfg.FlushInterval,

		OnError: func(_ context.Context, err error) {
			logging.For("es-indexer").Error("bulk indexer error", "error", err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	return &Indexer{
		bulk: bulk,

		index: index,
	}, nil
}

// Start 启动前置: 确保索引存在 ( 幂等 )。
func (i *Indexer) Start(ctx context.Context, client *elasticsearch.Client) error {
	return EnsureMessageIndex(ctx, client, i.index)
}

// Close 关闭 bulk indexer ( 排空缓冲 )。
func (i *Indexer) Close(_ context.Context) error {
	return i.bulk.Close(context.Background())
}

// ProcessIndexEvent 是索引事件的 Kafka 消费入口 ( xprocessor.MsgHandler )。
func (i *Indexer) ProcessIndexEvent(ctx context.Context, msg *xmq.Message) error {
	event := &domain.MessageIndexEvent{}
	if err := json.Unmarshal(msg.Val, event); err != nil {
		// 毒事件跳过 ( ES 可重建，不值得无限重投 )。
		logging.For("es-indexer").Error(
			"failed to unmarshal index event, skip",
			"val", msg.Val,
			"error", err,
		)
		return nil
	}

	if event.MessageID == "" || event.ConversationID == "" || event.SenderID == 0 {
		logging.For("es-indexer").Error(
			"index event missing required fields, skip",
			"message_id", event.MessageID,
			"conversation_id", event.ConversationID,
			"sender_id", event.SenderID,
		)
		return nil
	}

	doc, err := json.Marshal(event)
	if err != nil {
		logging.For("es-indexer").Error(
			"failed to marshal index document, skip",
			"message_id", event.MessageID,
			"error", err,
		)
		return nil
	}

	// _id = message_id: 重复消费幂等 ( 覆盖写 )。
	if err := i.bulk.Add(ctx, esutil.BulkIndexerItem{
		Action: "index",

		DocumentID: event.MessageID,
		Body:       jsonReader(doc),

		OnFailure: func(_ context.Context, _ esutil.BulkIndexerItem, resp esutil.BulkIndexerResponseItem, err error) {
			logging.For("es-indexer").Error(
				"bulk item failed",
				"message_id", resp.DocumentID,
				"status", resp.Status,
				"bulk_error", err,
			)
		},
	}); err != nil {
		return fmt.Errorf("failed to enqueue index event [ %s ]: %w", event.MessageID, err)
	}

	return nil
}

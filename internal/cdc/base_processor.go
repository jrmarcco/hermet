package cdc

import (
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

// ProcessorConfig 是 CDC 处理器的配置。
type ProcessorConfig struct {
	DBShardCount int // 数据库分片数量
	TBShardCount int // 表分片数量

	GroupID      string // 消费者组 ID
	TopicPattern string // 主题名称模式 ( 例如 "pgsql_db_%d.public.table_%d" )

	HandleTimeout time.Duration // 消息处理超时时间
}

var _ Processor = (*BaseProcessor)(nil)

// BaseProcessor 是 CDC 基础处理器，封装了分片消费的逻辑。
type BaseProcessor struct {
	*xprocessor.ConsumeProcessor
}

// NewBaseProcessor 创建一个新的 CDC 基础处理器。
func NewBaseProcessor(
	cfg ProcessorConfig,
	handler xprocessor.MsgHandler,
	factory xconsumer.ConsumerFactory,
) *BaseProcessor {
	topics := make([]xprocessor.TopicConfig, 0, cfg.DBShardCount*cfg.TBShardCount)
	for i := 0; i < cfg.DBShardCount; i++ {
		for j := 0; j < cfg.TBShardCount; j++ {
			topics = append(topics, xprocessor.TopicConfig{
				Topic:   fmt.Sprintf(cfg.TopicPattern, i, j),
				GroupID: cfg.GroupID,
			})
		}
	}

	consumerProcessor := xprocessor.NewConsumeProcessor(
		xprocessor.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       handler,
			HandleTimeout: cfg.HandleTimeout,
		},
		factory,
	)

	return &BaseProcessor{
		ConsumeProcessor: consumerProcessor,
	}
}

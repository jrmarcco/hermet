package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Consumer = (*DefaultConsumer)(nil)

type DefaultConsumer struct {
	consumer *xprocess.ConsumeProcessor
}

func NewDefaultConsumer(
	factory xconsume.ConsumerFactory,
	topic, groupID string,
	partitions int32,
	handler xprocess.MsgHandler,
	handleTimeout time.Duration,
) (*DefaultConsumer, error) {
	if topic == "" || groupID == "" || partitions <= 0 || handler == nil {
		return nil, fmt.Errorf("topic, groupID, partitions and handler are required")
	}

	// 为每个分区创建主题配置 ( Kafka 会自动分配分区 )。
	topics := make([]xprocess.TopicConfig, partitions)
	for i := range partitions {
		topics[i] = xprocess.TopicConfig{
			Topic:   topic,
			GroupID: groupID,
		}
	}

	// 创建消费处理器。
	consumer := xprocess.NewConsumeProcessor(
		xprocess.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       handler,
			HandleTimeout: handleTimeout,
		},
		factory,
	)

	return &DefaultConsumer{
		consumer: consumer,
	}, nil
}

func (c *DefaultConsumer) Start(ctx context.Context) error {
	if c.consumer != nil {
		return c.consumer.Start(ctx)
	}
	return nil
}

func (c *DefaultConsumer) Stop(ctx context.Context) error {
	if c.consumer != nil {
		return c.consumer.Stop(ctx)
	}
	return nil
}

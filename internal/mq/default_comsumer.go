package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

var _ Consumer = (*DefaultConsumer)(nil)

type DefaultConsumer struct {
	consumer *xprocessor.ConsumeProcessor
}

func NewDefaultConsumer(
	factory xconsumer.ConsumerFactory,
	topic, groupID string,
	partitions int32,
	handler xprocessor.MsgHandler,
	handleTimeout time.Duration,
) (*DefaultConsumer, error) {
	if topic == "" || groupID == "" || partitions <= 0 || handler == nil {
		return nil, fmt.Errorf("topic, groupID, partitions and handler are required")
	}

	// 为每个分区创建主题配置 ( Kafka 会自动分配分区 )。
	topics := make([]xprocessor.TopicConfig, partitions)
	for i := range partitions {
		topics[i] = xprocessor.TopicConfig{
			Topic:   topic,
			GroupID: groupID,
		}
	}

	// 创建消费处理器。
	consumer := xprocessor.NewConsumeProcessor(
		xprocessor.ConsumeProcessorConfig{
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

package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
)

var _ Processor = (*BizMsgProcessor)(nil)

// BizMsgProcessor 是业务消息处理器。
type BizMsgProcessor struct {
	consumer *xprocess.ConsumeProcessor
	producer *xprocess.ProduceProcessor

	factory xconsume.ConsumerFactory

	topic         string
	groupID       string
	partitions    int32
	handleTimeout time.Duration
}

// NewBizMsgProcessor 创建一个新的业务消息处理器。
func NewBizMsgProcessor(
	producer xproduce.Producer,
	factory xconsume.ConsumerFactory,
	topic, groupID string,
	partitions int32,
	handleTimeout time.Duration,
) (*BizMsgProcessor, error) {
	if topic == "" || groupID == "" || partitions <= 0 {
		return nil, fmt.Errorf("topic, groupID and partitions are required")
	}

	// 创建生产处理器。
	produceProcessor, err := xprocess.NewProduceProcessor(topic, producer)
	if err != nil {
		return nil, err
	}

	return &BizMsgProcessor{
		producer: produceProcessor,

		factory: factory,

		topic:         topic,
		groupID:       groupID,
		partitions:    partitions,
		handleTimeout: handleTimeout,
	}, nil
}

// StartConsumer 启动消费者，需要传入消息处理函数。
func (p *BizMsgProcessor) StartConsumer(ctx context.Context, handler xprocess.MsgHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	// 为每个分区创建主题配置 ( Kafka 会自动分配分区 )。
	topics := make([]xprocess.TopicConfig, p.partitions)
	for i := range p.partitions {
		topics[i] = xprocess.TopicConfig{
			Topic:   p.topic,
			GroupID: p.groupID,
		}
	}

	// 创建消费处理器。
	p.consumer = xprocess.NewConsumeProcessor(
		xprocess.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       handler,
			HandleTimeout: p.handleTimeout,
		},
		p.factory,
	)

	return p.consumer.Start(ctx)
}

// Start 启动处理器 ( 仅用于兼容 xprocess.Processor 接口 )。
func (p *BizMsgProcessor) Start(ctx context.Context) error {
	if p.consumer != nil {
		return p.consumer.Start(ctx)
	}
	return nil
}

// Stop 停止处理器。
func (p *BizMsgProcessor) Stop(ctx context.Context) error {
	if p.consumer != nil {
		return p.consumer.Stop(ctx)
	}
	return nil
}

// Produce 发送消息到主题。
func (p *BizMsgProcessor) Produce(ctx context.Context, key, val []byte) error {
	return p.producer.Produce(ctx, key, val)
}

package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
)

var _ Processor = (*GroupMemberProcessor)(nil)

// GroupMemberProcessor 是聊天群成员业务消息处理器。
type GroupMemberProcessor struct {
	consumeProcessor *xprocess.ConsumeProcessor
	produceProcessor *xprocess.ProduceProcessor

	factory xconsume.ConsumerFactory

	topic         string
	groupID       string
	partitions    int32
	handleTimeout time.Duration
}

// NewGroupMemberProcessor 创建一个新的聊天群成员处理器。
func NewGroupMemberProcessor(
	producer xproduce.Producer,
	factory xconsume.ConsumerFactory,
	topic, groupID string,
	partitions int32,
	handleTimeout time.Duration,
) (*GroupMemberProcessor, error) {
	if topic == "" || groupID == "" || partitions <= 0 {
		return nil, fmt.Errorf("topic, groupID and partitions are required")
	}

	// 创建生产处理器。
	produceProcessor, err := xprocess.NewProduceProcessor(topic, producer)
	if err != nil {
		return nil, err
	}

	return &GroupMemberProcessor{
		produceProcessor: produceProcessor,

		factory: factory,

		topic:         topic,
		groupID:       groupID,
		partitions:    partitions,
		handleTimeout: handleTimeout,
	}, nil
}

// StartConsumer 启动消费者，需要传入消息处理函数。
func (p *GroupMemberProcessor) StartConsumer(ctx context.Context, handler xprocess.MsgHandler) error {
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
	p.consumeProcessor = xprocess.NewConsumeProcessor(
		xprocess.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       handler,
			HandleTimeout: p.handleTimeout,
		},
		p.factory,
	)

	return p.consumeProcessor.Start(ctx)
}

// Start 启动处理器 ( 仅用于兼容 xprocess.Processor 接口)。
func (p *GroupMemberProcessor) Start(ctx context.Context) error {
	if p.consumeProcessor != nil {
		return p.consumeProcessor.Start(ctx)
	}
	return nil
}

// Stop 停止处理器。
func (p *GroupMemberProcessor) Stop(ctx context.Context) error {
	if p.consumeProcessor != nil {
		return p.consumeProcessor.Stop(ctx)
	}
	return nil
}

// Produce 发送消息到主题。
func (p *GroupMemberProcessor) Produce(ctx context.Context, key, val []byte) error {
	return p.produceProcessor.Produce(ctx, key, val)
}

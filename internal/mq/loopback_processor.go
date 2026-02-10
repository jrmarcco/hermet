package mq

import (
	"context"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
)

var _ Processor = (*LoopbackProcessor)(nil)

// LoopbackProcessor 是回环处理器 ( 即消费和生产同一个 Topic 的消息 )。
type LoopbackProcessor struct {
	consumer *DefaultConsumer
	producer *DefaultProducer
}

// NewLoopbackProcessor 创建一个新的回环处理器。
func NewLoopbackProcessor(
	producer xproduce.Producer,
	factory xconsume.ConsumerFactory,
	topic, groupID string,
	partitions int32,
	handler xprocess.MsgHandler,
	handleTimeout time.Duration,
) (*LoopbackProcessor, error) {
	dp, err := NewDefaultProducer(topic, producer)
	if err != nil {
		return nil, err
	}

	dc, err := NewDefaultConsumer(factory, topic, groupID, partitions, handler, handleTimeout)
	if err != nil {
		return nil, err
	}

	return &LoopbackProcessor{
		producer: dp,
		consumer: dc,
	}, nil
}

// Start 启动处理器 ( 仅用于兼容 xprocess.Processor 接口 )。
func (p *LoopbackProcessor) Start(ctx context.Context) error {
	return p.consumer.Start(ctx)
}

// Stop 停止处理器。
func (p *LoopbackProcessor) Stop(ctx context.Context) error {
	return p.consumer.Stop(ctx)
}

// Produce 发送消息到主题。
func (p *LoopbackProcessor) Produce(ctx context.Context, key, val []byte) error {
	return p.producer.Produce(ctx, key, val)
}

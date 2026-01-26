package cdc

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"go.uber.org/zap"
)

type ProcessorConfig struct {
	DBShardCount int
	TBShardCount int

	GroupID      string
	TopicPattern string
}

var _ Processor = (*BaseProcessor)(nil)

type BaseProcessor struct {
	cfg ProcessorConfig

	consumerFactory xconsumer.ConsumerFactory

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

func NewBaseProcessor(
	cfg ProcessorConfig,
	consumerFactory xconsumer.ConsumerFactory,
	logger *zap.Logger,
) *BaseProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &BaseProcessor{
		cfg: cfg,

		consumerFactory: consumerFactory,

		ctx:        ctx,
		cancelFunc: cancel,
		logger:     logger,
	}
}

func (p *BaseProcessor) Start(processFunc ProcessFunc) error {
	// 对每个数据库分片和表分片，创建一个消费者，并消费消息。
	for i := 0; i < p.cfg.DBShardCount; i++ {
		for j := 0; j < p.cfg.TBShardCount; j++ {
			topic := fmt.Sprintf(p.cfg.TopicPattern, i, j)
			consumer, err := p.consumerFactory.NewConsumer(topic, p.cfg.GroupID)
			if err != nil {
				p.logger.Error(
					"[hermet-cdc-base-processor] failed to create consumer",
					zap.String("topic", topic),
					zap.String("groupID", p.cfg.GroupID),
					zap.Error(err),
				)
				return err
			}

			msgChan, err := consumer.ConsumeChan(p.ctx)
			if err != nil {
				p.logger.Error(
					"[hermet-cdc-base-processor] failed to get message channel",
					zap.String("topic", topic),
					zap.String("groupID", p.cfg.GroupID),
					zap.Error(err),
				)
				return err
			}
			go p.consume(msgChan, processFunc)
		}
	}
	return nil
}

func (p *BaseProcessor) consume(msgChan <-chan *xmq.Message, processFunc ProcessFunc) {
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("[hermet-cdc-base-processor] processor context done")
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			if err := processFunc(p.ctx, msg); err != nil {
				p.logger.Error(
					"[hermet-cdc-base-processor] failed to process message",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
					zap.Int("partition", msg.Partition),
					zap.String("message", string(msg.Val)),
					zap.Error(err),
				)
			}
		}
	}
}

func (p *BaseProcessor) Stop() error {
	p.cancelFunc()
	return nil
}

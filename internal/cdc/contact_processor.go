package cdc

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"go.uber.org/zap"
)

var _ Processor = (*ContactProcessor)(nil)

// ContactProcessor 是联系人 CDC 处理器。
// 负责消费联系人 CDC 事件，并处理联系人相关业务。
type ContactProcessor struct {
	consumerFactory xconsumer.ConsumerFactory

	dbShardCount int // 数据库分片数量
	tbShardCount int // 表分片数量

	groupID string

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

func NewContactProcessor(
	consumerFactory xconsumer.ConsumerFactory,
	dbShardCount, tbShardCount int,
	groupID string,
	logger *zap.Logger,
) *ContactProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ContactProcessor{
		consumerFactory: consumerFactory,

		dbShardCount: dbShardCount,
		tbShardCount: tbShardCount,

		groupID: groupID,

		ctx:        ctx,
		cancelFunc: cancel,

		logger: logger,
	}
}

func (p *ContactProcessor) Start(processFunc ProcessFunc) error {
	// 对每个数据库分片和表分片，创建一个消费者，并消费消息。
	for i := 0; i < p.dbShardCount; i++ {
		for j := 0; j < p.tbShardCount; j++ {
			topic := fmt.Sprintf("pgsql_hermet_%d.public.contact_application_%d", i, j)

			consumer, err := p.consumerFactory.NewConsumer(topic, p.groupID)
			if err != nil {
				p.logger.Error(
					"[hermet-cdc-contact-processor] failed to create consumer",
					zap.String("topic", topic),
					zap.String("groupID", p.groupID),
					zap.Error(err),
				)
				return err
			}

			msgChan, err := consumer.ConsumeChan(p.ctx)
			if err != nil {
				p.logger.Error(
					"[hermet-cdc-contact-processor] failed to get message channel",
					zap.String("topic", topic),
					zap.String("groupID", p.groupID),
					zap.Error(err),
				)
				return err
			}

			go p.consume(msgChan, processFunc)
		}
	}

	return nil
}

func (p *ContactProcessor) consume(msgChan <-chan *xmq.Message, processFunc ProcessFunc) {
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("[hermet-cdc-contact-processor] processor context done")
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}

			if err := processFunc(p.ctx, msg); err != nil {
				p.logger.Error(
					"[hermet-cdc-contact-processor] failed to process message",
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

func (p *ContactProcessor) Stop() error {
	p.cancelFunc()
	return nil
}

package cdc

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"go.uber.org/zap"
)

var _ Processor = (*ConversationProcessor)(nil)

type ConversationProcessor struct {
	consumerFactory xconsumer.ConsumerFactory

	dbShardCount int // 数据库分片数量
	tbShardCount int // 表分片数量

	groupID string

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

func NewConversationProcessor(
	consumerFactory xconsumer.ConsumerFactory,
	dbShardCount, tbShardCount int,
	groupID string,
	zapLogger *zap.Logger,
) *ConversationProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConversationProcessor{
		consumerFactory: consumerFactory,

		dbShardCount: dbShardCount,
		tbShardCount: tbShardCount,

		groupID: groupID,

		ctx:        ctx,
		cancelFunc: cancel,

		logger: zapLogger,
	}
}

func (p *ConversationProcessor) Start(_ ProcessFunc) error {
	return nil
}

func (p *ConversationProcessor) Stop() error {
	p.cancelFunc()
	return nil
}

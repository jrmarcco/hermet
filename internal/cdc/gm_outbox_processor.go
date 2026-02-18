package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

var _ Processor = (*OutboxEventProcessor)(nil)

type OutboxEventProcessor struct {
	*BaseProcessor
}

func NewOutboxEventProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsumer.ConsumerFactory,
	handler xprocessor.MsgHandler,
) *OutboxEventProcessor {
	return &OutboxEventProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.outbox_event_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

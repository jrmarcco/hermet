package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*OutboxEventProcessor)(nil)

type OutboxEventProcessor struct {
	*BaseProcessor
}

func NewOutboxEventProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
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

package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*GmOutboxProcessor)(nil)

type GmOutboxProcessor struct {
	*BaseProcessor
}

func NewGmOutboxProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
) *GmOutboxProcessor {
	return &GmOutboxProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.group_member_outbox_event_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

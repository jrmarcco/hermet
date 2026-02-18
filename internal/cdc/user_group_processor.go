package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

type UserGroupProcessor struct {
	*BaseProcessor
}

func NewUserGroupProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsumer.ConsumerFactory,
	handler xprocessor.MsgHandler,
) *UserGroupProcessor {
	return &UserGroupProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.user_group_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

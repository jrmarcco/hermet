package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

type UserGroupProcessor struct {
	*BaseProcessor
}

func NewUserGroupProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
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

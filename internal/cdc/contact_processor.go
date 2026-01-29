package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*ContactProcessor)(nil)

// ContactProcessor 是联系人 CDC 处理器。
// 负责消费联系人 CDC 事件，当联系人申请审批通过后，建立用户之间双向关联关系。
type ContactProcessor struct {
	*BaseProcessor
}

// NewContactProcessor 创建一个新的联系人 CDC 处理器。
func NewContactProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
) *ContactProcessor {
	return &ContactProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.contact_application_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

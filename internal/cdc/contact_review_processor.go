package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

var _ Processor = (*ContactReviewProcessor)(nil)

// ContactReviewProcessor 是联系人审批 CDC 处理器。
// 负责消费联系人审批 CDC 事件。
// 当联系人申请审批通过后，建立用户之间双向关联关系。
type ContactReviewProcessor struct {
	*BaseProcessor
}

// NewContactReviewProcessor 创建一个新的联系人审批 CDC 处理器。
func NewContactReviewProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsumer.ConsumerFactory,
	handler xprocessor.MsgHandler,
) *ContactReviewProcessor {
	return &ContactReviewProcessor{
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

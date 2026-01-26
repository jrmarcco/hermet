package cdc

import (
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"go.uber.org/zap"
)

var _ Processor = (*ContactProcessor)(nil)

// ContactProcessor 是联系人 CDC 处理器。
// 负责消费联系人 CDC 事件，当联系人申请审批通过后，建立用户之间双向关联关系。
type ContactProcessor struct {
	*BaseProcessor
}

func NewContactProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	consumerFactory xconsumer.ConsumerFactory,
	logger *zap.Logger,
) *ContactProcessor {
	return &ContactProcessor{
		BaseProcessor: NewBaseProcessor(ProcessorConfig{
			DBShardCount: dbShardCount,
			TBShardCount: tbShardCount,
			GroupID:      groupID,
			TopicPattern: "pgsql_hermet_%d.public.contact_application_%d",
		}, consumerFactory, logger),
	}
}

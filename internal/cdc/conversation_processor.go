package cdc

import (
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"go.uber.org/zap"
)

var _ Processor = (*ConversationProcessor)(nil)

// ConversationProcessor 是会话 CDC 处理器。
// 负责消费会话 CDC 事件，当用户联系人关系建立后，创建会话。
type ConversationProcessor struct {
	*BaseProcessor
}

func NewConversationProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	consumerFactory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
) *ConversationProcessor {
	return &ConversationProcessor{
		BaseProcessor: NewBaseProcessor(ProcessorConfig{
			DBShardCount: dbShardCount,
			TBShardCount: tbShardCount,
			GroupID:      groupID,
			TopicPattern: "pgsql_hermet_%d.public.conversation_%d",
		}, consumerFactory, zapLogger),
	}
}

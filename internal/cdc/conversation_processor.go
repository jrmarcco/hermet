package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*ConversationProcessor)(nil)

// ConversationProcessor 是会话 CDC 处理器。
// 负责消费会话 CDC 事件，当用户联系人关系建立后，创建会话。
type ConversationProcessor struct {
	*BaseProcessor
}

// NewConversationProcessor 创建一个新的会话 CDC 处理器。
func NewConversationProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	handler xprocess.MsgHandler,
	factory xconsume.ConsumerFactory,
) *ConversationProcessor {
	return &ConversationProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.conversation_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

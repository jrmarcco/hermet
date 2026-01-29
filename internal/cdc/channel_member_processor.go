package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*ChannelMemberProcessor)(nil)

// ChannelMemberProcessor 是频道成员 CDC 处理器。
// 负责消费频道成员 CDC 事件。
// 当频道成员添加后，创建频道成员视图 ( 冗余成员信息 )。
type ChannelMemberProcessor struct {
	*BaseProcessor
}

// NewChannelMemberProcessor 创建一个新的频道成员 CDC 处理器。
func NewChannelMemberProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
) *ChannelMemberProcessor {
	return &ChannelMemberProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.channel_member_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

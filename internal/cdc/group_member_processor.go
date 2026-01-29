package cdc

import (
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

var _ Processor = (*GroupMemberProcessor)(nil)

// GroupMemberProcessor 是聊天群成员 CDC 处理器。
// 负责消费聊天群成员 CDC 事件。
// 当聊天群成员添加后，创建聊天群成员视图 ( 冗余成员信息 )。
type GroupMemberProcessor struct {
	*BaseProcessor
}

// NewGroupMemberProcessor 创建一个新的聊天群成员 CDC 处理器。
func NewGroupMemberProcessor(
	dbShardCount, tbShardCount int,
	groupID string,
	handleTimeout time.Duration,
	factory xconsume.ConsumerFactory,
	handler xprocess.MsgHandler,
) *GroupMemberProcessor {
	return &GroupMemberProcessor{
		BaseProcessor: NewBaseProcessor(
			ProcessorConfig{
				DBShardCount: dbShardCount,
				TBShardCount: tbShardCount,

				GroupID:      groupID,
				TopicPattern: "pgsql_hermet_%d.public.group_member_%d",

				HandleTimeout: handleTimeout,
			},
			handler,
			factory,
		),
	}
}

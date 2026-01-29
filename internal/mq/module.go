package mq

import (
	"context"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var MqFxModule = fx.Module(
	"mq",
	fx.Provide(
		newGroupMemberProcessor,
		fx.Annotate(
			newGroupMemberProcessor,
			fx.As(new(service.GroupMemberMsgProducer)),
		),
	),
	fx.Invoke(startGroupMemberConsumer),
)

func newGroupMemberProcessor(producer xproduce.Producer, factory xconsume.ConsumerFactory) (*GroupMemberProcessor, error) {
	type config struct {
		Topic         string        `mapstructure:"topic"`
		GroupID       string        `mapstructure:"group_id"`
		Partitions    int32         `mapstructure:"partitions"`
		HandleTimeout time.Duration `mapstructure:"handle_timeout"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("mq.group_member_add", &cfg); err != nil {
		return nil, err
	}

	processor, err := NewGroupMemberProcessor(
		producer,
		factory,

		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,
		cfg.HandleTimeout,
	)
	if err != nil {
		return nil, err
	}
	return processor, nil
}

// startGroupMemberConsumer 启动聊天群成员消费者 ( fx Invoke 函数 )。
func startGroupMemberConsumer(
	processor *GroupMemberProcessor,
	groupService service.GroupService,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.StartConsumer(ctx, groupService.ProcessAddMembersMsg); err != nil {
				logger.Error("failed to start group member consumer", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return processor.Stop(ctx)
		},
	})
}

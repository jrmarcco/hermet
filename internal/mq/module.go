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
		newChannelMemberProcessor,
		// 将 ChannelMemberProcessor 作为 ChannelMessageProducer 提供给 service 包。
		fx.Annotate(
			newChannelMemberProcessor,
			fx.As(new(service.ChannelMsgProducer)),
		),
	),
	fx.Invoke(startChannelMemberConsumer),
)

func newChannelMemberProcessor(producer xproduce.Producer, factory xconsume.ConsumerFactory) (*ChannelMemberProcessor, error) {
	type config struct {
		Topic         string        `mapstructure:"topic"`
		GroupID       string        `mapstructure:"group_id"`
		Partitions    int32         `mapstructure:"partitions"`
		HandleTimeout time.Duration `mapstructure:"handle_timeout"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("mq.channel_member_add", &cfg); err != nil {
		return nil, err
	}

	processor, err := NewChannelMemberProcessor(
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

// startChannelMemberConsumer 启动频道成员消费者 ( fx Invoke 函数 )。
func startChannelMemberConsumer(
	processor *ChannelMemberProcessor,
	channelService service.ChannelService,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.StartConsumer(ctx, channelService.ProcessAddMembersMsg); err != nil {
				logger.Error("failed to start channel member consumer", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return processor.Stop(ctx)
		},
	})
}

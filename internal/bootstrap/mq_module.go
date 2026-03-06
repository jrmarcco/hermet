package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproducer"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// mq_module.go 负责配置消息队列相关的依赖注入。

var MqFxModule = fx.Module(
	"mq",
	fx.Provide(
		fx.Annotate(
			newGroupMemberProducer,
			fx.As(new(mq.Producer)),
			fx.ResultTags(`name:"group_member_producer"`),
		),
		fx.Annotate(
			newDownwstreamMsgProducer,
			fx.As(new(mq.Producer)),
			fx.ResultTags(`name:"downstream_msg_producer"`),
		),
	),
	fx.Invoke(
		startGroupMemberConsumer,
		startUpstreamMsgConsumer,
	),
)

type mqConfig struct {
	Topic         string        `mapstructure:"topic"`
	GroupID       string        `mapstructure:"group_id"`
	Partitions    int32         `mapstructure:"partitions"`
	HandleTimeout time.Duration `mapstructure:"handle_timeout"`
}

// newGroupMemberProducer 创建一个聊天群成员消息生产者。
func newGroupMemberProducer(producer xproducer.Producer) (*mq.DefaultProducer, error) {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.producer.group_member_add", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group member producer config: %w", err)
	}

	gmp, err := mq.NewDefaultProducer(cfg.Topic, producer)
	if err != nil {
		return nil, err
	}

	return gmp, nil
}

// newDownwstreamMsgProducer 创建一个下游消息生产者。
func newDownwstreamMsgProducer(producer xproducer.Producer) (*mq.DefaultProducer, error) {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.producer.message_downstream", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal downstream message producer config: %w", err)
	}

	dp, err := mq.NewDefaultProducer(cfg.Topic, producer)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

type gmcFxParams struct {
	fx.In

	Factory      xconsumer.ConsumerFactory
	GroupService service.GroupService

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

// startGroupMemberConsumer 注册聊天群成员消费者生命周期。
func startGroupMemberConsumer(params gmcFxParams) error {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.consumer.group_member_add", &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal group member consumer config: %w", err)
	}

	consumer, err := mq.NewDefaultConsumer(
		params.Factory,
		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,
		params.GroupService.ProcessAddMembersMsg,
		cfg.HandleTimeout,
	)
	if err != nil {
		return err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := consumer.Start(ctx); err != nil {
				params.Logger.Error("failed to start group member consumer", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Stop(ctx)
		},
	})
	return nil
}

type umcFxParams struct {
	fx.In

	Factory        xconsumer.ConsumerFactory
	MessageService service.MessageService

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

// startUpstreamMsgConsumer 注册上游消息消费者生命周期。
func startUpstreamMsgConsumer(params umcFxParams) error {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.consumer.message_upstream", &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal upstream message consumer config: %w", err)
	}

	dc, err := mq.NewDefaultConsumer(
		params.Factory,

		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,

		params.MessageService.ProcessUpstreamMsg,
		cfg.HandleTimeout,
	)
	if err != nil {
		return err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := dc.Start(ctx); err != nil {
				params.Logger.Error("failed to start upstream message consumer", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := dc.Stop(ctx); err != nil {
				params.Logger.Error("failed to stop upstream message consumer", zap.Error(err))
				return err
			}
			return nil
		},
	})
	return nil
}

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproducer"
	"github.com/jrmarcco/hermet/internal/search"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
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
			newMessageNotifyProducer,
			fx.As(new(mq.Producer)),
			fx.ResultTags(`name:"message_notify_producer"`),
		),
		fx.Annotate(
			newMessageIndexProducer,
			fx.As(new(mq.Producer)),
			fx.ResultTags(`name:"message_index_producer"`),
		),
	),
	fx.Invoke(
		startGroupMemberConsumer,
		startUpstreamMsgConsumer,
		startMessageIndexConsumer,
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

// newMessageNotifyProducer 创建通知 topic 生产者。
// 消息落库后投递 NotifyMessage ( 仅携带 latest_seq )，推送侧消费后下发 NOTIFY 帧。
func newMessageNotifyProducer(producer xproducer.Producer) (*mq.DefaultProducer, error) {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.producer.message_notify", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message notify producer config: %w", err)
	}

	np, err := mq.NewDefaultProducer(cfg.Topic, producer)
	if err != nil {
		return nil, err
	}

	return np, nil
}

// newMessageIndexProducer 创建索引事件生产者。
// 消息落库后投递 MessageIndexEvent，ES indexer 消费后批量写索引。
func newMessageIndexProducer(producer xproducer.Producer) (*mq.DefaultProducer, error) {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.producer.message_index", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message index producer config: %w", err)
	}

	ip, err := mq.NewDefaultProducer(cfg.Topic, producer)
	if err != nil {
		return nil, err
	}

	return ip, nil
}

type micFxParams struct {
	fx.In

	Factory   xconsumer.ConsumerFactory
	Indexer   *search.Indexer
	Client    *elasticsearch.Client
	Lifecycle fx.Lifecycle
}

// startMessageIndexConsumer 注册 ES 索引事件消费者生命周期。
func startMessageIndexConsumer(params micFxParams) error {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.consumer.message_index", &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal message index consumer config: %w", err)
	}

	dc, err := mq.NewDefaultConsumer(
		params.Factory,

		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,

		params.Indexer.ProcessIndexEvent,
		cfg.HandleTimeout,
	)
	if err != nil {
		return err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 索引前置: 确保索引存在 ( 幂等 )。
			if err := params.Indexer.Start(ctx, params.Client); err != nil {
				logging.For("mq").Error("failed to ensure message index", "error", err)
				return err
			}

			if err := dc.Start(ctx); err != nil {
				logging.For("mq").Error("failed to start message index consumer", "error", err)
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := dc.Stop(ctx); err != nil {
				logging.For("mq").Error("failed to stop message index consumer", "error", err)
				return err
			}
			return params.Indexer.Close(ctx)
		},
	})
	return nil
}

type gmcFxParams struct {
	fx.In

	Factory      xconsumer.ConsumerFactory
	GroupService service.GroupService
	Lifecycle    fx.Lifecycle
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
				logging.For("mq").Error("failed to start group member consumer", "error", err)
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
	Lifecycle      fx.Lifecycle
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
				logging.For("mq").Error("failed to start upstream message consumer", "error", err)
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := dc.Stop(ctx); err != nil {
				logging.For("mq").Error("failed to stop upstream message consumer", "error", err)
				return err
			}
			return nil
		},
	})
	return nil
}

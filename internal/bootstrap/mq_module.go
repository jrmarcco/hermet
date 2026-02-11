package bootstrap

import (
	"context"
	"time"

	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
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
			newGroupMemberMqProcessor,
			fx.As(new(mq.Producer)),
			fx.ResultTags(`name:"group_member_producer"`),
		),
	),
)

type mqConfig struct {
	Topic         string        `mapstructure:"topic"`
	GroupID       string        `mapstructure:"group_id"`
	Partitions    int32         `mapstructure:"partitions"`
	HandleTimeout time.Duration `mapstructure:"handle_timeout"`
}

type gmpFxParams struct {
	fx.In

	Producer xproduce.Producer
	Factory  xconsume.ConsumerFactory

	GroupService service.GroupService

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

// newGroupMemberProcessor 创建一个新的聊天群成员处理器。
func newGroupMemberMqProcessor(params gmpFxParams) (*mq.LoopbackProcessor, error) {
	cfg := mqConfig{}
	if err := viper.UnmarshalKey("mq.group_member_add", &cfg); err != nil {
		return nil, err
	}

	gmp, err := mq.NewLoopbackProcessor(
		params.Producer,
		params.Factory,

		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,

		params.GroupService.ProcessAddMembersMsg,
		cfg.HandleTimeout,
	)
	if err != nil {
		return nil, err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := gmp.Start(ctx); err != nil {
				params.Logger.Error("failed to start group member consumer", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return gmp.Stop(ctx)
		},
	})
	return gmp, nil
}

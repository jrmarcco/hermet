package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ServiceFxModule = fx.Module(
	"service",
	fx.Provide(
		newAuthService,

		// CDC service
		fx.Annotate(NewDefaultCDCService, fx.As(new(CDCService))),

		// User service
		fx.Annotate(NewDefaultUserService, fx.As(new(UserService))),

		// Contact service
		fx.Annotate(NewDefaultContactService, fx.As(new(ContactService))),

		// Conversation service
		fx.Annotate(NewDefaultConversationService, fx.As(new(ConversationService))),

		// Channel service
		fx.Annotate(newChannelService, fx.As(new(ChannelService))),
	),
)

type authServiceFxParams struct {
	fx.In

	Repo repo.BizUserRepo

	AtManager xjwt.Manager[authv1.JwtPayload] `name:"access-token-manager"`
	RtManager xjwt.Manager[authv1.JwtPayload] `name:"refresh-token-manager"`
}

// newAuthService 作为适配器，将 fx 的参数结构体转换为普通的函数调用。
func newAuthService(p authServiceFxParams) AuthService {
	return NewDefaultAuthService(
		p.Repo,
		p.AtManager,
		p.RtManager,
	)
}

type channelServiceFxParams struct {
	fx.In

	ConsumerFactory xconsumer.ConsumerFactory
	ChannelRepo     repo.ChannelRepo
	Producer        xproduce.Producer

	Logger *zap.Logger

	Lifecycle fx.Lifecycle
}

// newChannelService 创建频道服务。
func newChannelService(params channelServiceFxParams) (ChannelService, error) {
	type config struct {
		MemberSyncAddTopic     string `mapstructure:"member_sync_add_topic"`
		MemberSyncAddGroupID   string `mapstructure:"member_sync_add_group_id"`
		MemberSyncAddPartition int    `mapstructure:"member_sync_add_partition"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("business.channel", &cfg); err != nil {
		return nil, err
	}

	svc, err := NewDefaultChannelService(
		params.ConsumerFactory,
		cfg.MemberSyncAddTopic,
		cfg.MemberSyncAddGroupID,
		cfg.MemberSyncAddPartition,

		params.ChannelRepo,
		params.Producer,

		params.Logger,
	)
	if err != nil {
		return nil, err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return svc.Stop()
		},
	})
	return svc, nil
}

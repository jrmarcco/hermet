package service

import (
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

// newChannelService 创建频道服务。
// ChannelService 预留了同步创建成员数量阈值的配置项，需要额外封装一个 new 方法。
func newChannelService(channelRepo repo.ChannelRepo, producer xproduce.Producer, logger *zap.Logger) (ChannelService, error) {
	type config struct {
		MemberSyncAddThreshold int    `mapstructure:"member_sync_add_threshold"`
		MemberSyncAddTopic     string `mapstructure:"member_sync_add_topic"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("business.channel", &cfg); err != nil {
		return nil, err
	}

	return NewDefaultChannelService(
		cfg.MemberSyncAddThreshold,
		cfg.MemberSyncAddTopic,

		channelRepo,
		producer,

		logger,
	), nil
}

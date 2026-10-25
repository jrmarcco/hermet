package service

import (
	"fmt"

	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
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

		// Group service
		fx.Annotate(newGroupService, fx.As(new(GroupService))),

		// Channel service
		fx.Annotate(NewDefaultChannelService, fx.As(new(ChannelService))),

		// Message service
		fx.Annotate(newMessageService, fx.As(new(MessageService))),
	),
)

type authServiceFxParams struct {
	fx.In

	Repo repo.UserRepo

	Signer    *auth.RS256Signer
	RtManager xjwt.Manager[authv1.JwtPayload] `name:"refresh-token-manager"`
}

// newAuthService 作为适配器，将 fx 的参数结构体转换为普通的函数调用。
func newAuthService(p authServiceFxParams) AuthService {
	return NewDefaultAuthService(
		p.Repo,
		p.Signer,
		p.RtManager,
	)
}

type groupServiceFxParams struct {
	fx.In

	Producer mq.Producer `name:"group_member_producer"`

	GroupRepo       repo.GroupRepo
	GroupMemberRepo repo.GroupMemberRepo

	Logger *zap.Logger
}

func newGroupService(params groupServiceFxParams) (*DefaultGroupService, error) {
	return NewDefaultGroupService(
		params.Producer,
		params.GroupRepo,
		params.GroupMemberRepo,
		params.Logger,
	)
}

type messageServiceFxParams struct {
	fx.In

	MessageRepo repo.MessageRepo
	ContactRepo repo.ContactRepo

	GroupMemberRepo repo.GroupMemberRepo
	UserGroupRepo   repo.UserGroupRepo

	NotifyProducer mq.Producer `name:"message_notify_producer"`
	NotifyClients  *client.Manager[messagev1.NotifyServiceClient]
	IndexProducer  mq.Producer `name:"message_index_producer"`

	Gen idgen.Generator

	Logger *zap.Logger
}

func newMessageService(params messageServiceFxParams) (MessageService, error) {
	fanoutCfg, err := loadMessageFanoutConfig()
	if err != nil {
		return nil, err
	}

	return NewDefaultMessageService(
		params.MessageRepo,
		params.ContactRepo,

		params.GroupMemberRepo,
		params.UserGroupRepo,

		params.NotifyProducer,
		params.NotifyClients,
		params.IndexProducer,

		*fanoutCfg,

		params.Gen,

		params.Logger,
	), nil
}

type messageFanoutConfig struct {
	FanoutInlineThreshold int `mapstructure:"fanout_inline_threshold"`
	FanoutBatchSize       int `mapstructure:"fanout_batch_size"`
}

func loadMessageFanoutConfig() (*MessageFanoutConfig, error) {
	cfg := &messageFanoutConfig{}
	if err := viper.UnmarshalKey("hermet.message", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message fanout config: %w", err)
	}

	if cfg.FanoutInlineThreshold <= 0 || cfg.FanoutBatchSize <= 0 {
		return nil, fmt.Errorf("hermet.message.fanout_inline_threshold and fanout_batch_size must be positive")
	}

	return &MessageFanoutConfig{
		InlineThreshold: cfg.FanoutInlineThreshold,
		BatchSize:       cfg.FanoutBatchSize,
	}, nil
}

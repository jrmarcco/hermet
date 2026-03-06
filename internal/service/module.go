package service

import (
	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
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
		fx.Annotate(NewDefaultMessageService, fx.As(new(MessageService))),
	),
)

type authServiceFxParams struct {
	fx.In

	Repo repo.UserRepo

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

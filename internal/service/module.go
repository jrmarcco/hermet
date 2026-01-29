package service

import (
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	"go.uber.org/fx"
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

		// Channel service
		fx.Annotate(NewDefaultChannelService, fx.As(new(ChannelService))),

		// Channel service
		fx.Annotate(NewDefaultGroupService, fx.As(new(GroupService))),
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

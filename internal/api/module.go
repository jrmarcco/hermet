package api

import (
	"go.uber.org/fx"

	"github.com/jrmarcco/hermet/internal/pkg/xgin"
)

var APIFxModule = fx.Module(
	"api",
	fx.Provide(
		// 认证 ( Auth ) API。
		fx.Annotate(
			NewAuthHandler,
			fx.As(new(xgin.RouteRegistry)),
			fx.ResultTags(`group:"api_registry"`),
		),

		// 联系人 ( Contact ) API。
		fx.Annotate(
			NewContactHandler,
			fx.As(new(xgin.RouteRegistry)),
			fx.ResultTags(`group:"api_registry"`),
		),

		// 聊天群 ( Chat Group ) API。
		fx.Annotate(
			NewGroupHandler,
			fx.As(new(xgin.RouteRegistry)),
			fx.ResultTags(`group:"api_registry"`),
		),

		// 会话 ( Channel ) API。
		fx.Annotate(
			NewChannelHandler,
			fx.As(new(xgin.RouteRegistry)),
			fx.ResultTags(`group:"api_registry"`),
		),

		// FIXME: 临时 api 接口。
		fx.Annotate(
			NewAdminHandler,
			fx.As(new(xgin.RouteRegistry)),
			fx.ResultTags(`group:"api_registry"`),
		),
	),
)

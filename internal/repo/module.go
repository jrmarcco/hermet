package repo

import (
	"go.uber.org/fx"
)

var RepoFxModule = fx.Module(
	"repo",
	fx.Provide(
		// biz user repo ( 用户 )
		fx.Annotate(
			NewDefaultUserRepo,
			fx.As(new(UserRepo)),
		),

		// contact application repo ( 联系人申请 )
		fx.Annotate(
			NewDefaultContactApplicationRepo,
			fx.As(new(ContactApplicationRepo)),
		),

		// user contact repo ( 用户联系人 )
		fx.Annotate(
			NewDefaultContactRepo,
			fx.As(new(ContactRepo)),
		),

		// group application repo ( 聊天群申请 )
		fx.Annotate(
			NewDefaultGroupApplicationRepo,
			fx.As(new(GroupApplicationRepo)),
		),
		// group repo ( 聊天群 )
		fx.Annotate(
			NewDefaultGroupRepo,
			fx.As(new(GroupRepo)),
		),

		// channel repo ( 用户频道 )
		fx.Annotate(
			NewDefaultChannelRepo,
			fx.As(new(ChannelRepo)),
		),

		// message repo ( 消息 )
		fx.Annotate(
			NewDefaultMessageRepo,
			fx.As(new(MessageRepo)),
		),
	),
)

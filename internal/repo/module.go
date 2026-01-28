package repo

import (
	"go.uber.org/fx"
)

var RepoFxModule = fx.Module(
	"repo",
	fx.Provide(
		// biz user repo ( 用户 )
		fx.Annotate(
			NewDefaultBizUserRepo,
			fx.As(new(BizUserRepo)),
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

		// channel application repo ( 频道申请 )
		fx.Annotate(
			NewDefaultChannelApplicationRepo,
			fx.As(new(ChannelApplicationRepo)),
		),
		// channel repo ( 频道 )
		fx.Annotate(
			NewDefaultChannelRepo,
			fx.As(new(ChannelRepo)),
		),

		// user conversation repo ( 用户会话 )
		fx.Annotate(
			NewDefaultConversationRepo,
			fx.As(new(ConversationRepo)),
		),

		// message repo ( 消息 )
		fx.Annotate(
			NewDefaultMessageRepo,
			fx.As(new(MessageRepo)),
		),
	),
)

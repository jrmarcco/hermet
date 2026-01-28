package dao

import "go.uber.org/fx"

var DaoFxModule = fx.Module(
	"dao",
	fx.Provide(
		fx.Annotate(
			NewDefaultBizUserDao,
			fx.As(new(BizUserDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"biz_user_shard_helper"`),
		),

		// 联系人申请。
		fx.Annotate(
			NewDefaultContactApplicationDao,
			fx.As(new(ContactApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_application_shard_helper"`),
		),
		// 频道申请。
		fx.Annotate(
			NewDefaultChannelApplicationDao,
			fx.As(new(ChannelApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"channel_application_shard_helper"`),
		),

		// 联系人。
		fx.Annotate(
			NewDefaultContactDao,
			fx.As(new(ContactDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"contact_shard_helper"`,
				`name:"contact_view_shard_helper"`,
				`name:"contact_reverse_index_shard_helper"`,
			),
		),
		// 频道。
		fx.Annotate(
			NewDefaultChannelDao,
			fx.As(new(ChannelDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"channel_shard_helper"`,
				`name:"channel_member_shard_helper"`,
				`name:"channel_member_view_shard_helper"`,
				`name:"channel_member_reverse_index_shard_helper"`,
			),
		),

		// 会话。
		fx.Annotate(
			NewDefaultConversationDao,
			fx.As(new(ConversationDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"conversation_view_shard_helper"`,
				`name:"conversation_reverse_index_shard_helper"`,
			),
		),

		// 消息 Dao ( MongoDB )。
		fx.Annotate(NewMongoMessageDao, fx.As(new(MessageDao))),
	),
)

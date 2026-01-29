package dao

import "go.uber.org/fx"

var DaoFxModule = fx.Module(
	"dao",
	fx.Provide(
		fx.Annotate(
			NewDefaultUserDao,
			fx.As(new(UserDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"user_shard_helper"`),
		),

		// 联系人申请。
		fx.Annotate(
			NewDefaultContactApplicationDao,
			fx.As(new(ContactApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_application_shard_helper"`),
		),
		// 频道申请。
		fx.Annotate(
			NewDefaultGroupApplicationDao,
			fx.As(new(GroupApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"group_application_shard_helper"`),
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
		// 聊天群。
		fx.Annotate(
			NewDefaultGroupDao,
			fx.As(new(GroupDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"group_shard_helper"`,
				`name:"group_member_shard_helper"`,
				`name:"group_member_view_shard_helper"`,
				`name:"group_member_reverse_index_shard_helper"`,
			),
		),

		// 频道。
		fx.Annotate(
			NewDefaultChannelDao,
			fx.As(new(ChannelDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"channel_shard_helper"`,
				`name:"channel_reverse_index_shard_helper"`,
			),
		),

		// 消息 Dao ( MongoDB )。
		fx.Annotate(NewMongoMessageDao, fx.As(new(MessageDao))),
	),
)

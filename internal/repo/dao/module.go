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

		// 联系人申请表。
		fx.Annotate(
			NewDefaultContactApplicationDao,
			fx.As(new(ContactApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_application_shard_helper"`),
		),
		// 频道申请表。
		fx.Annotate(
			NewDefaultChannelApplicationDao,
			fx.As(new(ChannelApplicationDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"channel_application_shard_helper"`),
		),

		// 用户联系人表。
		fx.Annotate(
			NewDefaultUserContactDao,
			fx.As(new(UserContactDao)),
			fx.ParamTags(
				`name:"db_sharding_clients"`,
				`name:"user_contact_shard_helper"`,
				`name:"user_contact_view_shard_helper"`,
				`name:"contact_reverse_index_shard_helper"`,
			),
		),

		// 消息 Dao ( MongoDB )。
		fx.Annotate(
			NewMongoMessageDao,
			fx.As(new(MessageDao)),
		),
	),
)

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
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_shard_helper"`),
		),
		// 联系人视图。
		fx.Annotate(
			NewDefaultContactViewDao,
			fx.As(new(ContactViewDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_view_shard_helper"`),
		),
		// 联系人反向索引。
		fx.Annotate(
			NewDefaultContactRidxDao,
			fx.As(new(ContactRidxDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"contact_reverse_index_shard_helper"`),
		),

		// 聊天群。
		fx.Annotate(
			NewDefaultGroupDao,
			fx.As(new(GroupDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"group_shard_helper"`),
		),
		// 聊天群成员。
		fx.Annotate(
			NewDefaultGroupMemberDao,
			fx.As(new(GroupMemberDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"group_member_shard_helper"`),
		),
		// 聊天群成员视图。
		fx.Annotate(
			NewDefaultGroupMemberViewDao,
			fx.As(new(GroupMemberViewDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"group_member_view_shard_helper"`),
		),
		// 聊天群成员 outbox 事件。
		fx.Annotate(
			NewDefaultGroupMemberOutboxDao,
			fx.As(new(GroupMemberOutboxDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"group_member_outbox_shard_helper"`),
		),

		// 用户群组。
		fx.Annotate(
			NewDefaultUserGroupDao,
			fx.As(new(UserGroupDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"user_group_shard_helper"`),
		),
		// 用户群组视图。
		fx.Annotate(
			NewDefaultUserGroupViewDao,
			fx.As(new(UserGroupViewDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"user_group_view_shard_helper"`),
		),
		// 用户群组反向索引。
		fx.Annotate(
			NewDefaultUserGroupRidxDao,
			fx.As(new(UserGroupRidxDao)),
			fx.ParamTags(`name:"db_sharding_clients"`, `name:"user_group_reverse_index_shard_helper"`),
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

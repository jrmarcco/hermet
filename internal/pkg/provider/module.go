package provider

import (
	"go.uber.org/fx"

	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproducer"
	"github.com/jrmarcco/jit/xpool"
)

var (
	ZapLoggerFxModule  = fx.Module("zap-logger", fx.Provide(newZapLogger))
	EtcdFxModule       = fx.Module("etcd", fx.Provide(newEtcdClient))
	RedisFxModule      = fx.Module("redis", fx.Provide(newRedisCmdable))
	GoCacheFxModule    = fx.Module("go-cache", fx.Provide(newGoCache))
	JwtManagerFxModule = fx.Module("jwt-manager", fx.Provide(newJwtManager))
)

var DBFxModule = fx.Module(
	"db",
	fx.Provide(
		fx.Annotate(
			newDBShardingClients,
			fx.ResultTags(`name:"db_sharding_clients"`),
		),
	),
)

var (
	KafkaFxModule         = fx.Module("kafka", fx.Provide(newKafkaClient))
	KafkaProducerFxModule = fx.Module(
		"kafka-producer",
		fx.Provide(
			fx.Annotate(
				xproducer.NewKafkaProducer,
				fx.As(new(xproducer.Producer)),
			),
		),
	)
	KafkaConsumerFxModule = fx.Module(
		"kafka-consumer",
		fx.Provide(
			fx.Annotate(
				xconsumer.NewKafkaConsumerFactory,
				fx.As(new(xconsumer.ConsumerFactory)),
			),
		),
	)
)

var XpoolFxModule = fx.Module(
	"xpool",
	fx.Provide(
		fx.Annotate(
			newBlockTaskPool,
			fx.As(new(xpool.TaskPool)),
		),
	),
)

var JwtHandlerFxModule = fx.Module(
	"jwt-handler", fx.Provide(
		fx.Annotate(
			newRedisSessionHandler,
			fx.As(new(xsession.Handler)),
		),
	),
)

var MiddlewareFxModule = fx.Module(
	"middleware",
	fx.Provide(
		fx.Annotate(
			newCorsBuilder,
			fx.As(new(xgin.HandlerFuncBuilder)),
			fx.ResultTags(`group:"api_middleware"`),
		),
		fx.Annotate(
			newJwtBuilder,
			fx.As(new(xgin.HandlerFuncBuilder)),
			fx.ResultTags(`group:"api_middleware"`),
		),
	),
)

var ShardingFxModule = fx.Module(
	"sharding",
	fx.Provide(
		newIDGen,
		fx.Annotate(
			newUserShardHelper,
			fx.ResultTags(`name:"user_shard_helper"`),
		),

		// 联系人申请。
		fx.Annotate(
			newContactApplicationShardHelper,
			fx.ResultTags(`name:"contact_application_shard_helper"`),
		),
		// 聊天群申请。
		fx.Annotate(
			newGroupApplicationShardHelper,
			fx.ResultTags(`name:"group_application_shard_helper"`),
		),

		// 联系人。
		fx.Annotate(
			newContactShardHelper,
			fx.ResultTags(`name:"contact_shard_helper"`),
		),
		// 联系人视图。
		fx.Annotate(
			newContactViewShardHelper,
			fx.ResultTags(`name:"contact_view_shard_helper"`),
		),
		// 联系人反向索引。
		fx.Annotate(
			newContactReverseIndexShardHelper,
			fx.ResultTags(`name:"contact_reverse_index_shard_helper"`),
		),

		// 聊天群。
		fx.Annotate(
			newGroupShardHelper,
			fx.ResultTags(`name:"group_shard_helper"`),
		),
		// 聊天群成员。
		fx.Annotate(
			newGroupMemberShardHelper,
			fx.ResultTags(`name:"group_member_shard_helper"`),
		),
		// 聊天群成员视图。
		fx.Annotate(
			newGroupMemberViewShardHelper,
			fx.ResultTags(`name:"group_member_view_shard_helper"`),
		),

		// 用户群组。
		fx.Annotate(
			newUserGroupShardHelper,
			fx.ResultTags(`name:"user_group_shard_helper"`),
		),
		// 用户群组视图。
		fx.Annotate(
			newUserGroupViewShardHelper,
			fx.ResultTags(`name:"user_group_view_shard_helper"`),
		),
		// 用户群组反向索引。
		fx.Annotate(
			newUserGroupReverseIndexShardHelper,
			fx.ResultTags(`name:"user_group_reverse_index_shard_helper"`),
		),

		// 聊天频道。
		fx.Annotate(
			newChannelShardHelper,
			fx.ResultTags(`name:"channel_shard_helper"`),
		),
		// 聊天频道反向索引。
		fx.Annotate(
			newChannelReverseIndexShardHelper,
			fx.ResultTags(`name:"channel_reverse_index_shard_helper"`),
		),
		// outbox 事件。
		fx.Annotate(
			newOutboxEventShardHelper,
			fx.ResultTags(`name:"outbox_event_shard_helper"`),
		),

		// 消息正本 ( 按 conversation_id 分片 )。
		fx.Annotate(
			newMessageShardHelper,
			fx.ResultTags(`name:"message_shard_helper"`),
		),
		// 会话 seq 分配 ( 按 conversation_id 分片，与 message 同布局 )。
		fx.Annotate(
			newConversationSeqShardHelper,
			fx.ResultTags(`name:"conversation_seq_shard_helper"`),
		),
		// 收件箱 ( seq 分配与条目同布局，共用一个 helper，按 user_id 分片 )。
		fx.Annotate(
			newInboxEntryShardHelper,
			fx.ResultTags(`name:"inbox_entry_shard_helper"`),
		),
	),
)

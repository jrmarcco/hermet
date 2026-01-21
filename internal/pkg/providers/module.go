package providers

import (
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
	"go.uber.org/fx"
)

var (
	ZapLoggerFxModule  = fx.Module("zap-logger", fx.Provide(newZapLogger))
	RedisFxModule      = fx.Module("redis", fx.Provide(newRedisCmdable))
	MongoFxModule      = fx.Module("mongo", fx.Provide(newMongoClient))
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
				xproduce.NewKafkaProducer,
				fx.As(new(xproduce.Producer)),
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
			newBizUserShardHelper,
			fx.ResultTags(`name:"biz_user_shard_helper"`),
		),

		// 联系人申请表。
		fx.Annotate(
			newContactApplicationShardHelper,
			fx.ResultTags(`name:"contact_application_shard_helper"`),
		),
		// 频道申请表。
		fx.Annotate(
			newChannelApplicationShardHelper,
			fx.ResultTags(`name:"channel_application_shard_helper"`),
		),

		// 用户联系人表。
		fx.Annotate(
			newUserContactShardHelper,
			fx.ResultTags(`name:"user_contact_shard_helper"`),
		),
		// 用户联系人视图表。
		fx.Annotate(
			newUserContactViewShardHelper,
			fx.ResultTags(`name:"user_contact_view_shard_helper"`),
		),
		// 联系人反向索引表。
		fx.Annotate(
			newContactReverseIndexShardHelper,
			fx.ResultTags(`name:"contact_reverse_index_shard_helper"`),
		),
	),
)

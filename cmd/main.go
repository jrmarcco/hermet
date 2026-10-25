package main

import (
	"fmt"

	"github.com/jrmarcco/hermet/internal/api"
	"github.com/jrmarcco/hermet/internal/app"
	"github.com/jrmarcco/hermet/internal/bootstrap"
	"github.com/jrmarcco/hermet/internal/grpc"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/hermet/internal/pkg/provider"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/hermet/internal/search"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}

	// 日志装配必须先于 fx：fx 的事件日志在装配最早期求值。
	cleanupLogging := provider.SetupLogging()
	defer cleanupLogging()

	fx.New(
		// fx 事件日志与业务日志共用同一条 slog 管道。
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logging.For("fx")}
		}),

		// 初始化 etcd client。
		provider.EtcdFxModule,

		// 初始化 redis client。
		provider.RedisFxModule,

		// 初始化 go cache。
		provider.GoCacheFxModule,

		// 初始化 db。
		provider.DBFxModule,

		// 初始化 kafka client。
		provider.KafkaFxModule,
		provider.KafkaProducerFxModule,
		provider.KafkaConsumerFxModule,

		// 初始化 elasticsearch client ( 消息搜索查询侧 )。
		provider.EsClientFxModule,

		// 初始化网关 NotifyService 客户端 ( 即时通知通道 )。
		provider.NotifyClientFxModule,

		// 初始化 jwt manager。
		provider.JwtManagerFxModule,

		// 初始化 jwt handler。
		provider.JwtHandlerFxModule,

		// 初始化 middleware。
		provider.MiddlewareFxModule,

		// 初始化 sharding。
		provider.ShardingFxModule,

		// 初始化 gRPC 服务注册器与服务实例 ( etcd 服务发现 )。
		provider.GrpcRegistryFxModule,
		provider.GrpcServiceInstanceFxModule,

		// 初始化 xpool。
		provider.XpoolFxModule,

		// 初始化 dao。
		dao.DaoFxModule,

		// 初始化 repo。
		repo.RepoFxModule,

		// 初始化 service。
		service.ServiceFxModule,

		// 初始化 web。
		api.APIFxModule,

		// 初始化 gRPC server ( SyncService，网关经服务发现调用 )。
		grpc.GrpcFxModule,

		// 初始化搜索 ( ES indexer + 搜索服务 )。
		search.SearchFxModule,

		// 初始化运维管理端口 ( /metrics + /debug/pprof )。
		provider.AdminFxModule,

		// 初始化 cdc。
		bootstrap.CdcFxModule,

		// 初始化 mq。
		bootstrap.MqFxModule,

		// 初始化 app。
		app.AppFxModule,
	).Run()
}

func loadConfig() error {
	configFile := pflag.String("config", "etc/config.yaml", "config file (default: etc/config.yaml)")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	return nil
}

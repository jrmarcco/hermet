package main

import (
	"fmt"

	"github.com/jrmarcco/hermet/internal/api"
	"github.com/jrmarcco/hermet/internal/app"
	"github.com/jrmarcco/hermet/internal/bootstrap"
	"github.com/jrmarcco/hermet/internal/pkg/provider"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}

	fx.New(
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),

		// 初始化 zap.logger。
		provider.ZapLoggerFxModule,

		// 初始化 redis client。
		provider.RedisFxModule,

		// 初始化 db。
		provider.DBFxModule,

		// 初始化 mongo db client。
		provider.MongoFxModule,

		// 初始化 kafka client。
		provider.KafkaFxModule,
		provider.KafkaProducerFxModule,
		provider.KafkaConsumerFxModule,

		// 初始化 jwt manager。
		provider.JwtManagerFxModule,

		// 初始化 jwt handler。
		provider.JwtHandlerFxModule,

		// 初始化 middleware。
		provider.MiddlewareFxModule,

		// 初始化 sharding。
		provider.ShardingFxModule,

		// 初始化 dao。
		dao.DaoFxModule,

		// 初始化 repo。
		repo.RepoFxModule,

		// 初始化 service。
		service.ServiceFxModule,

		// 初始化 web。
		api.APIFxModule,

		// 初始化 cdc。
		bootstrap.CdcFxModule,

		// 初始化 mq。
		bootstrap.MqFxModule,

		// 初始化 app。
		app.AppFxModule,
	).Run()
}

func loadConfig() error {
	viper.AddConfigPath("etc")
	viper.SetConfigType("yaml")

	// 读取基础配置。
	viper.SetConfigName("base")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read base config: %w", err)
	}

	subConfigNames := []string{"redis", "db", "mongodb", "kafka", "sharding", "mq"}
	for _, subConfigName := range subConfigNames {
		viper.SetConfigName(subConfigName)
		if err := viper.MergeInConfig(); err != nil {
			return fmt.Errorf("failed to merge %s config: %w", subConfigName, err)
		}
	}

	return nil
}

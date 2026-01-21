package cdc

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var CdcFxModule = fx.Module(
	"cdc",
	fx.Invoke(newContactProcessor),
	fx.Invoke(newConversationProcessor),
)

type cdcProcessorConfig struct {
	GroupID      string `mapstructure:"group_id"`
	DBShardCount int    `mapstructure:"db_shard_count"`
	TBShardCount int    `mapstructure:"tb_shard_count"`
}

func newContactProcessor(
	cdcService service.CDCService,
	consumerFactory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*ContactProcessor, error) {
	cfg, err := loadCdcProcessorConfig("cdc.contact_processor")
	if err != nil {
		return nil, err
	}

	processor := NewContactProcessor(consumerFactory, cfg.DBShardCount, cfg.TBShardCount, cfg.GroupID, zapLogger)
	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if err := processor.Start(cdcService.ProcessContact); err != nil {
				zapLogger.Error("failed to start contact processor", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			return processor.Stop()
		},
	})
	return processor, nil
}

func newConversationProcessor(
	consumerFactory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*ConversationProcessor, error) {
	cfg, err := loadCdcProcessorConfig("cdc.conversation_processor")
	if err != nil {
		return nil, err
	}

	processor := NewConversationProcessor(consumerFactory, cfg.DBShardCount, cfg.TBShardCount, cfg.GroupID, zapLogger)
	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// TODO: 消费会话 CDC 事件，并处理会话相关业务。
			return nil
		},
		OnStop: func(_ context.Context) error {
			return processor.Stop()
		},
	})
	return processor, nil
}

func loadCdcProcessorConfig(key string) (cdcProcessorConfig, error) {
	cfg := cdcProcessorConfig{}
	if err := viper.UnmarshalKey(key, &cfg); err != nil {
		return cdcProcessorConfig{}, err
	}
	return cfg, nil
}

package cdc

import (
	"context"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/service"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var CdcFxModule = fx.Module(
	"cdc",
	fx.Invoke(newContactReviewProcessor),
	fx.Invoke(newContactProcessor),
	fx.Invoke(newChannelMemberProcessor),
)

type cdcProcessorConfig struct {
	GroupID       string        `mapstructure:"group_id"`
	HandleTimeout time.Duration `mapstructure:"handle_timeout"`

	DBShardCount int `mapstructure:"db_shard_count"`
	TBShardCount int `mapstructure:"tb_shard_count"`
}

func newContactReviewProcessor(
	cdcService service.CDCService,
	factory xconsume.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*ContactReviewProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.contact_review_processor")
	if err != nil {
		return nil, err
	}

	processor := NewContactReviewProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessContactReview,
	)
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start contact review processor", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return processor.Stop(ctx)
		},
	})
	return processor, nil
}

func newContactProcessor(
	cdcService service.CDCService,
	factory xconsume.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*ContactProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.contact")
	if err != nil {
		return nil, err
	}

	processor := NewContactProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessContact,
	)
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start contact processor", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return processor.Stop(ctx)
		},
	})
	return processor, nil
}

func newChannelMemberProcessor(
	cdcService service.CDCService,
	factory xconsume.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*ChannelMemberProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.channel_member_processor")
	if err != nil {
		return nil, err
	}

	processor := NewChannelMemberProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessChannelMember,
	)
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start channel member processor", zap.Error(err))
				return err
			}
			return nil
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

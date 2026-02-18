package bootstrap

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrmarcco/hermet/internal/cdc"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/service"
)

// cdc_module.go 负责配置 CDC ( Change Data Capture ) 相关的依赖注入。

var CdcFxModule = fx.Module(
	"cdc",
	fx.Invoke(newContactReviewCdcProcessor),
	fx.Invoke(newContactCdcProcessor),
	fx.Invoke(newUserGroupCdcProcessor),
	fx.Invoke(newGroupMemberCdcProcessor),
	fx.Invoke(newOutboxEventCdcProcessor),
)

type cdcProcessorConfig struct {
	GroupID       string        `mapstructure:"group_id"`
	HandleTimeout time.Duration `mapstructure:"handle_timeout"`

	DBShardCount int `mapstructure:"db_shard_count"`
	TBShardCount int `mapstructure:"tb_shard_count"`
}

func newContactReviewCdcProcessor(
	cdcService service.CDCService,
	factory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*cdc.ContactReviewProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.contact_review")
	if err != nil {
		return nil, err
	}

	processor := cdc.NewContactReviewProcessor(
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

func newContactCdcProcessor(
	cdcService service.CDCService,
	factory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*cdc.ContactProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.contact")
	if err != nil {
		return nil, err
	}

	processor := cdc.NewContactProcessor(
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

func newUserGroupCdcProcessor(
	cdcService service.CDCService,
	factory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*cdc.UserGroupProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.user_group")
	if err != nil {
		return nil, err
	}

	processor := cdc.NewUserGroupProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessUserGroup,
	)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start user group processor", zap.Error(err))
				return err
			}
			return nil
		},
	})
	return processor, nil
}

func newGroupMemberCdcProcessor(
	cdcService service.CDCService,
	factory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*cdc.GroupMemberProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.group_member")
	if err != nil {
		return nil, err
	}

	processor := cdc.NewGroupMemberProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessGroupMember,
	)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start group member processor", zap.Error(err))
				return err
			}
			return nil
		},
	})
	return processor, nil
}

func newOutboxEventCdcProcessor(
	cdcService service.CDCService,
	factory xconsumer.ConsumerFactory,
	zapLogger *zap.Logger,
	lifecycle fx.Lifecycle,
) (*cdc.OutboxEventProcessor, error) {
	cfg, err := loadCdcProcessorConfig("mq.cdc.outbox_event")
	if err != nil {
		return nil, err
	}

	processor := cdc.NewOutboxEventProcessor(
		cfg.DBShardCount,
		cfg.TBShardCount,
		cfg.GroupID,
		cfg.HandleTimeout,

		factory,
		cdcService.ProcessOutboxEvent,
	)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := processor.Start(ctx); err != nil {
				zapLogger.Error("failed to start group member outbox processor", zap.Error(err))
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

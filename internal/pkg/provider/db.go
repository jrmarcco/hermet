package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/jit/xsync"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jrmarcco/hermet/internal/pkg/xgorm"
)

func newDBShardingClients(zapLogger *zap.Logger, lc fx.Lifecycle) (*xsync.Map[string, *gorm.DB], error) {
	type shardingConfig struct {
		Name string `mapstructure:"name"`
		DSN  string `mapstructure:"dsn"`
	}

	type dbConfig struct {
		LogLevel                  string           `mapstructure:"log_level"`
		SlowThreshold             time.Duration    `mapstructure:"slow_threshold"`
		IgnoreRecordNotFoundError bool             `mapstructure:"ignore_record_not_found_error"`
		Sharding                  []shardingConfig `mapstructure:"sharding"`
	}

	cfg := dbConfig{}
	if err := viper.UnmarshalKey("db", &cfg); err != nil {
		return nil, err
	}

	var level logger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		level = logger.Silent
	case "error":
		level = logger.Error
	case "warn":
		level = logger.Warn
	case "info":
		level = logger.Info
	default:
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	// 用于错误时清理。
	openedDBs := make([]*gorm.DB, 0, len(cfg.Sharding))

	var dbs xsync.Map[string, *gorm.DB]
	for _, s := range cfg.Sharding {
		db, err := gorm.Open(postgres.Open(s.DSN), &gorm.Config{
			Logger: xgorm.NewZapLogger(
				zapLogger,
				xgorm.WithLogLevel(level),
				xgorm.WithSlowThreshold(cfg.SlowThreshold),
				xgorm.WithIgnoreRecordNotFoundError(cfg.IgnoreRecordNotFoundError),
			),
		})
		if err != nil {
			// 清理已打开的连接。
			closeOpenedDBs(openedDBs)
			return nil, fmt.Errorf("failed to open db [ %s ]: %w", s.Name, err)
		}

		innerDB, err := db.DB()
		if err != nil {
			// 清理已打开的连接。
			closeOpenedDBs(openedDBs)
			return nil, fmt.Errorf("failed to get sql db [ %s ]: %w", s.Name, err)
		}

		openedDBs = append(openedDBs, db)
		dbs.Store(s.Name, db)

		// 注册关闭 Hook。
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				if err := innerDB.Close(); err != nil {
					zapLogger.Error("[ioc-db] failed to close db connection", zap.Error(err))
					return fmt.Errorf("failed to close db connection: %w", err)
				}
				zapLogger.Info("[ioc-db] db connection closed")
				return nil
			},
		})
	}
	return &dbs, nil
}

func closeOpenedDBs(openedDBs []*gorm.DB) {
	for _, openedDB := range openedDBs {
		if db, _ := openedDB.DB(); db != nil {
			_ = db.Close()
		}
	}
}

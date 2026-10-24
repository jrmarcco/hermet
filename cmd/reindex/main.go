// reindex 命令: 从 PG message 表全量扫描重建 ES 消息索引。
//
// 用法 ( 在 backend 目录下 ):
//
//	go run ./cmd/reindex --config etc/config.yaml
//
// 语义: 广播所有 message 分片表，按雪花 id 游标分页，_id = message_id 覆盖写——
// 重复执行幂等，可与在线索引链路并行。
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/pkg/provider"
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen/snowflake"
	"github.com/jrmarcco/hermet/internal/search"
	"github.com/jrmarcco/jit/xsync"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	configFile := pflag.String("config", "etc/config.yaml", "config file (default: etc/config.yaml)")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	defer func() {
		_ = logger.Sync()
	}()

	esCfg, err := loadEsConfig()
	if err != nil {
		return err
	}

	// 手动装配 ( 工具型命令不走 fx ):
	// PG 分片连接 + message shard helper + ES client。
	dbs, err := newShardingDBs()
	if err != nil {
		return err
	}

	msgShard, err := newMessageShardHelper()
	if err != nil {
		return err
	}

	esClient, err := newEsClient(esCfg)
	if err != nil {
		return err
	}

	reindexer := search.NewReindexer(dbs, msgShard, esClient, esCfg.MessageIndex, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	start := time.Now()
	total, err := reindexer.Run(ctx)
	if err != nil {
		return fmt.Errorf("reindex failed ( indexed %d docs ): %w", total, err)
	}

	logger.Info("reindex done",
		zap.Int64("total", total),
		zap.Duration("elapsed", time.Since(start)),
	)
	return nil
}

func loadEsConfig() (*provider.EsConfig, error) {
	cfg := &provider.EsConfig{}
	if err := viper.UnmarshalKey("es", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal es config: %w", err)
	}
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("es.addresses is required")
	}
	if cfg.MessageIndex == "" {
		return nil, fmt.Errorf("es.message_index is required")
	}
	return cfg, nil
}

func newEsClient(cfg *provider.EsConfig) (*elasticsearch.Client, error) {
	client, err := elasticsearch.New(
		elasticsearch.WithAddresses(cfg.Addresses...),
		elasticsearch.WithBasicAuth(cfg.Username, cfg.Password),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}
	return client, nil
}

func newShardingDBs() (*xsync.Map[string, *gorm.DB], error) {
	type shardingDSN struct {
		Name string `mapstructure:"name"`
		DSN  string `mapstructure:"dsn"`
	}

	var shards []shardingDSN
	if err := viper.UnmarshalKey("db.sharding", &shards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db.sharding config: %w", err)
	}

	dbs := &xsync.Map[string, *gorm.DB]{}
	for _, shard := range shards {
		db, err := gorm.Open(postgres.Open(shard.DSN), &gorm.Config{
			Logger: nil,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect database [ %s ]: %w", shard.Name, err)
		}
		dbs.Store(shard.Name, db)
	}
	return dbs, nil
}

func newMessageShardHelper() (*sharding.ShardHelper, error) {
	cfg := struct {
		DBPrefix     string `mapstructure:"db_prefix"`
		TBPrefix     string `mapstructure:"tb_prefix"`
		DBShardCount uint64 `mapstructure:"db_shard_count"`
		TBShardCount uint64 `mapstructure:"tb_shard_count"`
	}{}
	if err := viper.UnmarshalKey("sharding.message", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sharding.message config: %w", err)
	}

	strategy, err := sharding.NewModuloSharding(snowflake.NewExtractor(), cfg.DBPrefix, cfg.TBPrefix, cfg.DBShardCount, cfg.TBShardCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create modulo sharding: %w", err)
	}

	helper, err := sharding.NewShardHelper(snowflake.NewGenerator(), strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create shard helper: %w", err)
	}
	return helper, nil
}

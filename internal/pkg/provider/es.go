package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// EsClientFxModule 提供 Elasticsearch 客户端与配置。
// ES 只是异步查询加速层 ( PG 是唯一事实源 )，客户端不可用时
// 不阻塞主服务启动 ( 搜索 API 报错兜底，索引 consumer 自行重试 )。
var EsClientFxModule = fx.Module(
	"es-client",
	fx.Provide(newEsConfig, newEsClient),
)

// EsConfig ES 配置 ( indexer 与搜索服务共用 )。
type EsConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`

	MessageIndex string `mapstructure:"message_index"`

	BulkNumWorkers    int           `mapstructure:"bulk_num_workers"`
	BulkFlushBytes    int           `mapstructure:"bulk_flush_bytes"`
	BulkFlushInterval time.Duration `mapstructure:"bulk_flush_interval"`

	RequestTimeout time.Duration `mapstructure:"request_timeout"`
}

func newEsConfig() (*EsConfig, error) {
	cfg := &EsConfig{}
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

func newEsClient(cfg *EsConfig, logger *zap.Logger, lifecycle fx.Lifecycle) (*elasticsearch.Client, error) {
	client, err := elasticsearch.New(
		elasticsearch.WithAddresses(cfg.Addresses...),
		elasticsearch.WithBasicAuth(cfg.Username, cfg.Password),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// 启动时探活: ES 不可达只告警不失败 ( 加速层可后置启动，reindex 可补数据 )。
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
			defer cancel()

			resp, pingErr := client.Info(client.Info.WithContext(pingCtx))
			if pingErr != nil {
				logger.Warn("[es-client] elasticsearch not reachable at startup, search/index will retry", zap.Error(pingErr))
				return nil
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			logger.Info("[es-client] elasticsearch connected", zap.Strings("addresses", cfg.Addresses))
			return nil
		},
	})

	return client, nil
}

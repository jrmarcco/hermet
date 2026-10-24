package search

import (
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/pkg/provider"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// SearchFxModule 提供 ES 索引器与搜索服务。
var SearchFxModule = fx.Module(
	"search",
	fx.Provide(
		newIndexer,
		newSearchService,
	),
)

func newIndexer(
	client *elasticsearch.Client,
	cfg *provider.EsConfig,
	logger *zap.Logger,
) (*Indexer, error) {
	return NewIndexer(
		client,
		cfg.MessageIndex,

		BulkConfig{
			NumWorkers:    cfg.BulkNumWorkers,
			FlushBytes:    cfg.BulkFlushBytes,
			FlushInterval: cfg.BulkFlushInterval,
		},

		logger,
	)
}

func newSearchService(
	client *elasticsearch.Client,
	cfg *provider.EsConfig,
	userGroupRepo repo.UserGroupRepo,
	logger *zap.Logger,
) *Service {
	return NewService(
		client,
		userGroupRepo,
		cfg.MessageIndex,
		logger,
	)
}

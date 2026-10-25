package search

import (
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/pkg/provider"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/fx"
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
) (*Indexer, error) {
	return NewIndexer(
		client,
		cfg.MessageIndex,

		BulkConfig{
			NumWorkers:    cfg.BulkNumWorkers,
			FlushBytes:    cfg.BulkFlushBytes,
			FlushInterval: cfg.BulkFlushInterval,
		},
	)
}

func newSearchService(
	client *elasticsearch.Client,
	cfg *provider.EsConfig,
	userGroupRepo repo.UserGroupRepo,
) *Service {
	return NewService(
		client,
		userGroupRepo,
		cfg.MessageIndex,
	)
}

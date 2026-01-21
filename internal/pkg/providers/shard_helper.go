package providers

import (
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen"
	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen/snowflake"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type idGenFxResult struct {
	fx.Out

	Gen       idgen.Generator
	Extractor sharding.ShardValExtractor
}

func newIDGen() (idGenFxResult, error) {
	gen := snowflake.NewGenerator()
	extractor := snowflake.NewExtractor()
	return idGenFxResult{
		Gen:       gen,
		Extractor: extractor,
	}, nil
}

type shardingConfig struct {
	DBPrefix     string `mapstructure:"db_prefix"`
	TBPrefix     string `mapstructure:"tb_prefix"`
	DBShardCount uint64 `mapstructure:"db_shard_count"`
	TBShardCount uint64 `mapstructure:"tb_shard_count"`
}

func newBizUserShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.biz_user", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newContactApplicationShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.contact_application", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newChannelApplicationShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.channel_application", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newUserContactShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.user_contact", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newUserContactViewShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.user_contact_view", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newContactReverseIndexShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.contact_reverse_index", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newShardHelper(
	gen idgen.Generator,
	extractor sharding.ShardValExtractor,
	cfg shardingConfig,
) (*sharding.ShardHelper, error) {
	strategy, err := sharding.NewModuloSharding(extractor, cfg.DBPrefix, cfg.TBPrefix, cfg.DBShardCount, cfg.TBShardCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create modulo sharding: %w", err)
	}

	helper, err := sharding.NewShardHelper(gen, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create shard helper: %w", err)
	}

	return helper, nil
}

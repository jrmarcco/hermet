package provider

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

func newUserShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
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

func newGroupApplicationShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.group_application", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newContactShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.contact", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newContactViewShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.contact_view", &cfg); err != nil {
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

func newGroupShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.chat_group", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newGroupMemberShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.group_member", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newGroupMemberViewShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.group_member_view", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newUserGroupShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.user_group", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newUserGroupViewShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.user_group_view", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newUserGroupReverseIndexShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.user_group_reverse_index", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newChannelShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.channel", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newChannelReverseIndexShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.channel_reverse_index", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newOutboxEventShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.outbox_event", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newMessageShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.message", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newConversationSeqShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	// conversation_seq 与 message 同布局 ( 按 conversation_id 路由 )。
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.conversation_seq", &cfg); err != nil {
		return nil, err
	}
	return newShardHelper(gen, extractor, cfg)
}

func newInboxEntryShardHelper(gen idgen.Generator, extractor sharding.ShardValExtractor) (*sharding.ShardHelper, error) {
	// inbox_seq 与 inbox_entry 同布局 ( 按 user_id 路由 )，共用本 helper。
	cfg := shardingConfig{}
	if err := viper.UnmarshalKey("sharding.inbox_entry", &cfg); err != nil {
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

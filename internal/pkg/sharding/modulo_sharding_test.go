package sharding

import (
	"testing"

	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen/snowflake"
	"github.com/stretchr/testify/require"
)

// 测试用分片前缀与期望值 ( goconst: 多个用例复用 )。
const (
	testDBPrefix = "db"
	testTBPrefix = "table"

	wantFirstDB = "db_0"
	wantFirstTB = "table_0"
)

func TestNewModuloSharding(t *testing.T) {
	t.Parallel()

	extractor := snowflake.NewExtractor()
	tests := []struct {
		name         string
		extractor    ShardValExtractor
		dbPrefix     string
		tbPrefix     string
		dbShardCount uint64
		tbShardCount uint64
		wantErr      bool
	}{
		{
			name:         "valid parameters",
			extractor:    extractor,
			dbPrefix:     testDBPrefix,
			tbPrefix:     testTBPrefix,
			dbShardCount: 8,
			tbShardCount: 4,
			wantErr:      false,
		},
		{
			name:         "nil extractor",
			extractor:    nil,
			dbPrefix:     testDBPrefix,
			tbPrefix:     testTBPrefix,
			dbShardCount: 8,
			tbShardCount: 4,
			wantErr:      true,
		},
		{
			name:         "zero db shard count",
			extractor:    extractor,
			dbPrefix:     testDBPrefix,
			tbPrefix:     testTBPrefix,
			dbShardCount: 0,
			tbShardCount: 4,
			wantErr:      true,
		},
		{
			name:         "zero tb shard count",
			extractor:    extractor,
			dbPrefix:     testDBPrefix,
			tbPrefix:     testTBPrefix,
			dbShardCount: 8,
			tbShardCount: 0,
			wantErr:      true,
		},
		{
			name:         "empty db prefix",
			extractor:    extractor,
			dbPrefix:     "",
			tbPrefix:     testTBPrefix,
			dbShardCount: 8,
			tbShardCount: 4,
			wantErr:      true,
		},
		{
			name:         "empty tb prefix",
			extractor:    extractor,
			dbPrefix:     testDBPrefix,
			tbPrefix:     "",
			dbShardCount: 8,
			tbShardCount: 4,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewModuloSharding(
				tt.extractor, tt.dbPrefix, tt.tbPrefix, tt.dbShardCount, tt.tbShardCount,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewModuloSharding() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModuloSharding_Shard(t *testing.T) {
	t.Parallel()

	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "db", "table", 8, 4)
	if err != nil {
		t.Fatalf("NewModuloSharding() error = %v", err)
	}

	tests := []struct {
		name     string
		shardVal uint64
		wantDB   string
		wantTB   string
	}{
		{
			name:     "shard val 0",
			shardVal: 0,
			wantDB:   wantFirstDB,
			wantTB:   wantFirstTB,
		},
		{
			name:     "shard val 1",
			shardVal: 1,
			wantDB:   "db_1",
			wantTB:   wantFirstTB,
		},
		{
			name:     "shard val 8",
			shardVal: 8,
			wantDB:   wantFirstDB,
			wantTB:   "table_1",
		},
		{
			name:     "shard val 31",
			shardVal: 31,
			wantDB:   "db_7",
			wantTB:   "table_3",
		},
		{
			name:     "shard val 32 (overflow, same as 0)",
			shardVal: 32,
			wantDB:   wantFirstDB,
			wantTB:   wantFirstTB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst, err := strategy.Shard(tt.shardVal)
			require.NoError(t, err)

			if dst.DB != tt.wantDB {
				t.Errorf("Shard() DB = %v, want %v", dst.DB, tt.wantDB)
			}
			if dst.TB != tt.wantTB {
				t.Errorf("Shard() TB = %v, want %v", dst.TB, tt.wantTB)
			}
		})
	}
}

func TestModuloSharding_UniformDistribution(t *testing.T) {
	t.Parallel()

	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "db", "table", 8, 4)
	if err != nil {
		t.Fatalf("NewModuloSharding() error = %v", err)
	}

	// 统计分布。
	distribution := make(map[string]int)
	totalShards := uint64(32)

	for i := range 1000 {
		dst, err := strategy.Shard(uint64(i))
		require.NoError(t, err)

		key := dst.DB + "." + dst.TB
		distribution[key]++
	}

	// 验证分布均匀性。
	// 1000 个值分到 32 个分片，期望每个分片约 31 个。
	expectedPerShard := 1000 / int(totalShards)
	tolerance := 5 // 允许 ±5 的误差。

	for key, count := range distribution {
		if count < expectedPerShard-tolerance || count > expectedPerShard+tolerance {
			t.Errorf(
				"Distribution not uniform: %s has %d items, expected ~%d",
				key, count, expectedPerShard,
			)
		}
	}

	// 验证所有分片都被使用。
	if len(distribution) != int(totalShards) {
		t.Errorf(
			"Not all shards used: got %d shards, want %d",
			len(distribution), totalShards,
		)
	}
}

func TestModuloSharding_DstFromId(t *testing.T) {
	t.Parallel()

	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "db", "table", 8, 4)
	if err != nil {
		t.Fatalf("NewModuloSharding() error = %v", err)
	}

	gen := snowflake.NewGenerator()

	// 生成 ID 并验证能正确提取分片信息。
	for i := range 100 {
		shardVal := uint64(i % 1024)

		expectedDst, err := strategy.Shard(shardVal)
		require.NoError(t, err)

		id, err := gen.NextID(shardVal)
		require.NoError(t, err)

		actualDst, err := strategy.DstFromID(id)
		require.NoError(t, err)

		if actualDst.DB != expectedDst.DB || actualDst.TB != expectedDst.TB {
			t.Errorf(
				"DstFromId() = %v.%v, want %v.%v",
				actualDst.DB, actualDst.TB, expectedDst.DB, expectedDst.TB,
			)
		}
	}
}

func TestModuloSharding_Broadcast(t *testing.T) {
	t.Parallel()

	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "db", "table", 4, 2)
	if err != nil {
		t.Fatalf("NewModuloSharding() error = %v", err)
	}

	dsts := strategy.Broadcast()

	// 验证数量：4 个库 * 2 张表 = 8 个分片。
	expectedCount := 4 * 2
	if len(dsts) != expectedCount {
		t.Errorf("Broadcast() returned %d shards, want %d", len(dsts), expectedCount)
	}

	// 验证没有重复。
	seen := make(map[string]bool)
	for _, dst := range dsts {
		key := dst.DB + "." + dst.TB
		if seen[key] {
			t.Errorf("Broadcast() contains duplicate: %s", key)
		}
		seen[key] = true
	}

	// 验证所有分片都存在。
	for i := range 4 {
		for j := range 2 {
			expected, err := strategy.Shard(uint64(i*2 + j))
			require.NoError(t, err)

			key := expected.DB + "." + expected.TB
			if !seen[key] {
				t.Errorf("Broadcast() missing shard: %s", key)
			}
		}
	}
}

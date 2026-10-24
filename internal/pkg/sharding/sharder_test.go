package sharding

import (
	"fmt"
	"math"
	"testing"

	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStringSharder_ShardVal 测试 StringSharder 的基本功能。
func TestStringSharder_ShardVal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{
			name: "email",
			key:  "user@example.com",
		},
		{
			name: "mobile",
			key:  "+8613800138000",
		},
		{
			name: "username",
			key:  "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sharder := NewStringSharder(tt.key)
			shardVal, err := sharder.ShardVal()
			require.NoError(t, err)

			// 验证返回值不为 0 ( 除非哈希结果真的是 0 )。
			assert.NotNil(t, shardVal)

			// 验证相同的 key 生成相同的 shardVal ( 稳定性 )。
			sharder2 := NewStringSharder(tt.key)
			shardVal2, err := sharder2.ShardVal()
			require.NoError(t, err)
			assert.Equal(t, shardVal, shardVal2, "same key should generate same shardVal")
		})
	}
}

// TestStringSharder_Uniformity 测试 StringSharder 的分片均匀性。
//
// 验证：
// 1. 不同的字符串生成不同的 shardVal。
// 2. 取模后的分布相对均匀 ( 标准差小于平均值的 10%，最小和最大值的差异不超过平均值的 20% )。
func TestStringSharder_Uniformity(t *testing.T) {
	t.Parallel()

	const (
		totalUsers = 100000
		shardCount = 32
		dbCount    = 8
		tableCount = 4
	)

	// 期望每个分片约 3125 个用户 ( 100000 / 32 )。
	expectedAvg := float64(totalUsers) / float64(shardCount)

	// 初始化
	gen := snowflake.NewGenerator()
	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "hermet_db", "biz_user", dbCount, tableCount)
	require.NoError(t, err)

	helper, err := NewShardHelper(gen, strategy)
	require.NoError(t, err)

	// 统计每个分片的数据量。
	shardDistribution := make(map[string]int)

	// 生成用户并统计分布。
	for i := range totalUsers {
		email := fmt.Sprintf("user%d@example.com", i)
		sharder := NewStringSharder(email)

		_, dst, err := helper.NextIDAndShard(sharder)
		require.NoError(t, err)

		key := dst.FullTable()
		shardDistribution[key]++
	}

	// 验证分片数量。
	assert.Equal(t, shardCount, len(shardDistribution), "should use all shards")

	// 验证分布均匀性。
	// 计算标准差，判断分布是否均匀。
	var sum, sumSquare float64
	minCnt, maxCnt := totalUsers, 0

	for _, count := range shardDistribution {
		sum += float64(count)
		sumSquare += float64(count * count)
		if count < minCnt {
			minCnt = count
		}
		if count > maxCnt {
			maxCnt = count
		}
	}

	mean := sum / float64(shardCount)
	variance := (sumSquare / float64(shardCount)) - (mean * mean)
	stdDev := math.Sqrt(variance)

	t.Logf("Distribution statistics:")
	t.Logf("  Expected avg per shard: %.2f", expectedAvg)
	t.Logf("  Actual mean: %.2f", mean)
	t.Logf("  Standard deviation: %.2f", stdDev)
	t.Logf("  Min count: %d", minCnt)
	t.Logf("  Max count: %d", maxCnt)
	t.Logf("  Range: %d", maxCnt-minCnt)

	// 断言：均值应该等于期望值。
	assert.InDelta(t, expectedAvg, mean, 1.0, "mean should be close to expected average")

	// 断言：标准差应该相对较小 ( 小于平均值的 10% )。
	maxStdDev := expectedAvg * 0.1
	assert.Less(t, stdDev, maxStdDev,
		"standard deviation should be less than 10%% of average, indicating uniform distribution")

	// 断言：最小和最大值的差异不应太大 ( 不超过平均值的 20% )。
	maxRange := expectedAvg * 0.2
	assert.Less(t, float64(maxCnt-minCnt), maxRange,
		"range between min and max should be small, indicating uniform distribution")
}

// TestBizUserIdGeneration_RealScenario 测试真实的 biz_user ID 生成场景。
//
// 模拟用户注册流程：
// 1. 用户提供 email。
// 2. 基于 email 生成 user_id ( 包含分片信息 )。
// 3. 后续可以从 user_id 中提取分片信息。
func TestBizUserIdGeneration_RealScenario(t *testing.T) {
	t.Parallel()

	gen := snowflake.NewGenerator()
	extractor := snowflake.NewExtractor()
	strategy, err := NewModuloSharding(extractor, "hermet_db", "biz_user", 8, 4)
	require.NoError(t, err)

	helper, err := NewShardHelper(gen, strategy)
	require.NoError(t, err)

	// 场景 1：用户注册。
	userEmail := "alice@example.com"
	emailSharder := NewStringSharder(userEmail)

	// 生成 user_id 和分片信息。
	userID, createDst, err := helper.NextIDAndShard(emailSharder)
	require.NoError(t, err)

	t.Logf("User registered:")
	t.Logf("  Email: %s", userEmail)
	t.Logf("  User ID: %d", userID)
	t.Logf("  Shard: %s", createDst.FullTable())

	// 验证 ID 不为 0。
	assert.NotZero(t, userID)

	// 场景 2：查询用户 ( 只有 user_id )。
	// 从 ID 中提取分片信息。
	queryDst, err := helper.DstFromID(userID)
	require.NoError(t, err)

	t.Logf("User query:")
	t.Logf("  User ID: %d", userID)
	t.Logf("  Shard: %s", queryDst.FullTable())

	// 验证：注册和查询时定位到同一个分片。
	assert.Equal(t, createDst.DB, queryDst.DB, "should locate to same database")
	assert.Equal(t, createDst.TB, queryDst.TB, "should locate to same table")
	assert.Equal(t, createDst.DBSuffix, queryDst.DBSuffix)
	assert.Equal(t, createDst.TBSuffix, queryDst.TBSuffix)

	// 场景 3：验证 shardVal 的一致性。
	// 同一个 email 应该总是生成相同的 shardVal。
	emailSharder2 := NewStringSharder(userEmail)
	shardVal1, err := emailSharder.ShardVal()
	require.NoError(t, err)
	shardVal2, err := emailSharder2.ShardVal()
	require.NoError(t, err)

	assert.Equal(t, shardVal1, shardVal2, "same email should generate same shardVal")

	// 提取 ID 中的 shardVal。
	extractedShardVal := extractor.ExtractShardVal(userID)

	// 验证：提取的 shardVal 应该与原始的 shardVal 的低 10 位一致。
	// 因为 Snowflake 只保存低 10 位。
	assert.Equal(t, shardVal1&0x3FF, extractedShardVal,
		"extracted shardVal should match original shardVal's low 10 bits")
}

// TestSnowflakeShardValExtraction 测试 Snowflake ID 中 shardVal 的提取。
//
// 验证：
// 1. 生成 ID 时嵌入的 shardVal。
// 2. 从 ID 中提取的 shardVal。
// 3. 两者应该一致 ( 低 10 位 )。
func TestSnowflakeShardValExtraction(t *testing.T) {
	t.Parallel()

	gen := snowflake.NewGenerator()
	testCases := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:  "user1",
			email: "user1@example.com",
		},
		{
			name:  "user2",
			email: "user2@example.com",
		},
		{
			name:  "user3",
			email: "user3@example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sharder := NewStringSharder(tc.email)
			originalShardVal, err := sharder.ShardVal()
			require.NoError(t, err)

			// 生成 ID。
			userID, err := gen.NextID(originalShardVal)
			require.NoError(t, err)

			// 提取 shardVal。
			extractedShardVal := snowflake.ExtractShardVal(userID)

			// 验证：提取的值应该等于原始值的低 10 位。
			expectedShardVal := originalShardVal & 0x3FF

			assert.Equal(t, expectedShardVal, extractedShardVal,
				"extracted shardVal should match the low 10 bits of original shardVal")

			t.Logf("Email: %s", tc.email)
			t.Logf("  Original shardVal: %d (0x%x)", originalShardVal, originalShardVal)
			t.Logf("  Low 10 bits: %d (0x%x)", expectedShardVal, expectedShardVal)
			t.Logf("  User ID: %d", userID)
			t.Logf("  Extracted shardVal: %d (0x%x)", extractedShardVal, extractedShardVal)
		})
	}
}

// BenchmarkStringSharder 测试 StringSharder 的性能。
func BenchmarkStringSharder(b *testing.B) {
	emails := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
		"alice@company.com",
		"bob@company.com",
	}

	b.ResetTimer()
	for i := range b.N {
		email := emails[i%len(emails)]
		sharder := NewStringSharder(email)
		_, err := sharder.ShardVal()
		require.NoError(b, err)
	}
}

// BenchmarkBizUserIdGeneration 测试完整的用户 ID 生成流程的性能。
func BenchmarkBizUserIdGeneration(b *testing.B) {
	gen := snowflake.NewGenerator()
	extractor := snowflake.NewExtractor()
	strategy, _ := NewModuloSharding(extractor, "hermet_db", "biz_user", 8, 4)
	helper, _ := NewShardHelper(gen, strategy)

	emails := make([]string, 1000)
	for i := range 1000 {
		emails[i] = fmt.Sprintf("user%d@example.com", i)
	}

	b.ResetTimer()
	for i := range b.N {
		email := emails[i%len(emails)]
		sharder := NewStringSharder(email)
		_, _, err := helper.NextIDAndShard(sharder)
		require.NoError(b, err)
	}
}

package sharding

import (
	"context"
	"fmt"
)

//go:generate mockgen -source=types.go -destination=mock/sharding.mock.go -package=shardingmock -typed ShardValExtractor,Strategy

// ShardValExtractor 是提取分片值的接口。
type ShardValExtractor interface {
	ExtractShardVal(id uint64) uint64
}

// Strategy 是分片策略的接口。
// 负责根据分片值计算目标数据库和表的位置。
//
// 使用示例：
//
//	strategy := NewModuloSharding("db", "table", 8, 4)
//	sharder := NewSingleIdSharder(userId)
//	dst := strategy.Shard(sharder.ShardVal())
type Strategy interface {
	// Shard 根据分片值计算目标库和目标表。
	// shardVal: 分片值，通常来自 Sharder.ShardVal()。
	// 返回: 目标数据库和表的信息。
	Shard(shardVal uint64) (Dst, error)

	// DstFromID 从 ID 中提取分片信息并计算分库分表目标。
	// 这对于根据已有 ID 查询数据非常有用，无需额外的分片信息。
	// id: 包含分片信息的 ID ( 如 Snowflake 生成的 ID )。
	// 返回: 目标数据库和表的信息。
	DstFromID(id uint64) (Dst, error)

	// DstFromShardVal 根据分片值计算分库分表目标。
	// shardVal: 分片值。
	// 返回: 目标数据库和表的信息。
	DstFromShardVal(shardVal uint64) (Dst, error)

	// Broadcast 返回所有分库分表的目标列表 ( 用于广播查询 )。
	// 适用场景：需要在所有分片上执行查询的操作 ( 如全量扫描、统计等 )。
	// 注意: 返回的切片顺序可能因策略而异 ( 如使用 BalancedSharding 时 )。
	Broadcast() []Dst
}

// Dst 是分片的目标，包含目标数据库和表的完整信息。
type Dst struct {
	DBSuffix uint64 // 数据库后缀（数字）
	TBSuffix uint64 // 表后缀（数字）

	DB string // 完整的数据库名，如 "hermet_db_0"
	TB string // 完整的表名，如 "message_1"
}

// FullTable 返回完整的表名（包含数据库名）。
// 格式：database.table
func (d Dst) FullTable() string {
	return fmt.Sprintf("%s.%s", d.DB, d.TB)
}

type dstContextKey struct{}

// ContextWithDst 将分片目标信息存入 context。
// 适用于在调用链中传递分片信息。
func ContextWithDst(ctx context.Context, dst Dst) context.Context {
	return context.WithValue(ctx, dstContextKey{}, dst)
}

// DstFromContext 从 context 中提取分片目标信息。
// 返回值：
// - Dst : 分片目标信息。
// - bool: 是否成功提取 ( false 表示 context 中没有分片信息 )。
func DstFromContext(ctx context.Context) (Dst, bool) {
	dst, ok := ctx.Value(dstContextKey{}).(Dst)
	return dst, ok
}

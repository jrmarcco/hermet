package sharding

import (
	"errors"
	"math/rand/v2"
)

// BroadcastMode 广播模式。
type BroadcastMode string

const (
	BroadcastModeDefault    BroadcastMode = "default"     // 默认模式：保持原有的广播顺序
	BroadcastModeRoundRobin BroadcastMode = "round_robin" // 轮询模式：交替轮询不同数据库的表
	BroadcastModeShuffle    BroadcastMode = "shuffle"     // 随机打乱模式：随机打乱所有分片顺序
)

var _ Strategy = (*BalancedSharding)(nil)

// BalancedSharding 负载均衡分片策略（装饰器模式）。
// 它包装一个基础分片策略，并为 Broadcast 方法提供负载均衡能力。
// Shard 和 DstFromId 方法直接委托给基础策略。
//
// 使用场景：
// - 当需要在所有分片上执行查询时（Broadcast），希望均衡各个数据库的负载
// - 避免某些数据库在广播查询时承受过重的连续负载
//
// 支持的负载均衡模式：
// - default: 保持原有顺序（相当于不做负载均衡）
// - round_robin: 轮询模式，交替访问不同数据库的表
// - shuffle: 随机打乱所有分片的访问顺序
//
// 示例：
//
//	baseStrategy, _ := NewModuloSharding("db", "table", 4, 2)
//	balancedStrategy, _ := NewBalancedSharding(baseStrategy, BroadcastModeRoundRobin)
//	dsts := balancedStrategy.Broadcast()
//	// 返回的 dsts 会按轮询顺序排列，避免连续访问同一个数据库
type BalancedSharding struct {
	base Strategy      // 基础分片策略
	mode BroadcastMode // 广播模式
}

// NewBalancedSharding 创建负载均衡分片策略。
// - base: 基础分片策略，不能为 nil
// - mode: 广播模式，可选值：default、round_robin、shuffle
func NewBalancedSharding(base Strategy, mode BroadcastMode) (*BalancedSharding, error) {
	if base == nil {
		return nil, errors.New("base strategy cannot be nil")
	}

	// 验证广播模式
	switch mode {
	case BroadcastModeDefault, BroadcastModeRoundRobin, BroadcastModeShuffle:
		// 合法的模式
	case "":
		// 空字符串默认为 default 模式
		mode = BroadcastModeDefault
	default:
		return nil, errors.New("invalid broadcast mode")
	}

	return &BalancedSharding{
		base: base,
		mode: mode,
	}, nil
}

func (s *BalancedSharding) Shard(shardVal uint64) (Dst, error) {
	return s.base.Shard(shardVal)
}

func (s *BalancedSharding) DstFromID(id uint64) (Dst, error) {
	return s.base.DstFromID(id)
}

func (s *BalancedSharding) DstFromShardVal(shardVal uint64) (Dst, error) {
	return s.base.DstFromShardVal(shardVal)
}

func (s *BalancedSharding) Broadcast() []Dst {
	dsts := s.base.Broadcast()

	switch s.mode {
	case BroadcastModeRoundRobin:
		return s.roundRobinBroadcast(dsts)
	case BroadcastModeShuffle:
		return s.shuffleBroadcast(dsts)
	default:
		// 默认保持原有的广播顺序
		return dsts
	}
}

// roundRobinBroadcast 轮询模式的广播。
// 将分片按数据库分组，然后交替轮询不同数据库的表。
// 例如：[db0.t0, db1.t0, db2.t0, db0.t1, db1.t1, db2.t1, ...]
// 这样可以避免连续访问同一个数据库，均衡数据库负载。
func (s *BalancedSharding) roundRobinBroadcast(dsts []Dst) []Dst {
	if len(dsts) == 0 {
		return dsts
	}

	var dbs []string
	dbGroup := make(map[string][]Dst)

	// 按数据库分组
	for _, dst := range dsts {
		if _, ok := dbGroup[dst.DB]; !ok {
			dbs = append(dbs, dst.DB)
		}
		dbGroup[dst.DB] = append(dbGroup[dst.DB], dst)
	}

	res := make([]Dst, 0, len(dsts))

	// 找出表数量最多的数据库
	maxTbCnt := 0
	for _, tbs := range dbGroup {
		if len(tbs) > maxTbCnt {
			maxTbCnt = len(tbs)
		}
	}

	// 轮询：每次从每个数据库取一张表
	for i := 0; i < maxTbCnt; i++ {
		for _, db := range dbs {
			if i < len(dbGroup[db]) {
				res = append(res, dbGroup[db][i])
			}
		}
	}

	return res
}

// shuffleBroadcast 随机打乱模式的广播。
// 将所有分片随机打乱后返回。
// 适用于希望每次广播查询的访问顺序都不同的场景，进一步分散负载。
// 使用 Go 1.20+ 的 math/rand/v2，自动初始化随机种子。
func (s *BalancedSharding) shuffleBroadcast(dsts []Dst) []Dst {
	if len(dsts) == 0 {
		return dsts
	}

	res := make([]Dst, len(dsts))
	copy(res, dsts)

	// 使用 rand.Shuffle 进行随机打乱。
	rand.Shuffle(len(res), func(i, j int) { //nolint:gosec // G404: 打乱分片列表用于负载均衡，非加密场景
		res[i], res[j] = res[j], res[i]
	})

	return res
}

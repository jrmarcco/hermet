package snowflake

import (
	"errors"
	"sync"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/sharding/idgen"
)

const (
	timestampBits = 41
	hashBits      = 10
	sequenceBits  = 12

	shardValShift  = sequenceBits // 分片值的偏移量
	timestampShift = shardValShift + hashBits

	sequenceMask  = (uint64(1) << sequenceBits) - 1
	hashMask      = (uint64(1) << hashBits) - 1
	timestampMask = (uint64(1) << timestampBits) - 1

	// epoch time 设置成 2025-01-01 00:00:00
	epochMillis   = uint64(1735689600000) // milliseconds of 2025-01-01 00:00:00
	number1000    = uint64(1000)
	number1000000 = uint64(1000000)

	// maxBackwardMillis 最大允许的时钟回拨毫秒数 ( 5s )
	maxBackwardMillis = 5000
)

var _ idgen.Generator = (*Generator)(nil)

// Generator 是自定义的雪花算法 id 生成器。
// 其中机器码的 10 位替换为分片值 ( shard value )。
//
// ┌──────────────────────────────────────────────────────────────────┐
// │    41 bit timestamp   │  10 bit shardVal   │   12 bit sequence   │
// └──────────────────────────────────────────────────────────────────┘
//
// 分片值可以是：
// - 由业务 id 和业务 key 计算而来的 hash 值
// - 单个业务标识的 hash 值
// - 固定的机器 ID
// - ...
// 通过 id 能解析出分片值从而获得分库分表信息。
//
// 并发安全：Generator 内部使用互斥锁保证并发安全。
// 注意：shardVal 参数只有低 10 位会被使用，高位会被截断。
type Generator struct {
	mu sync.Mutex

	sequence uint64
	lastTime uint64 // 上一次生成 id 的时间 ( 毫秒 )

	epoch time.Time
}

func NewGenerator() *Generator {
	return &Generator{
		sequence: 0,
		lastTime: 0,
		epoch:    time.Unix(int64(epochMillis/number1000), int64((epochMillis%number1000)*number1000000)),
	}
}

// NextID 生成 id。
//
// id 组成信息:
// ├── 41 位时间戳，基准时间为 2025-01-01 00:00:00。
// ├── 10 位分片值 ( 用于分库分表 )。
// ├── 12 位自增序列。
//
// 参数 shardVal 将被截取低 10 位嵌入到 ID 中。
//
// 并发安全：此方法是并发安全的。
// 时钟回拨：如果检测到时钟回拨且小于 5 秒，会等待时钟追上。
//
// 如果超过 5 秒，会 panic ( 需要人工介入处理 )。
//
// 序列号溢出：同一毫秒内如果序列号用完 ( >4095 )，会等待下一毫秒。
func (g *Generator) NextID(shardVal uint64) (uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	timestamp := uint64(time.Now().UnixMilli()) - epochMillis

	// 处理时钟回拨。
	if timestamp < g.lastTime {
		backwardMillis := g.lastTime - timestamp
		if backwardMillis > maxBackwardMillis {
			// 时钟回拨超过阈值，这是严重问题，需要人工介入。
			return 0, errors.New("clock moved backwards too much, please check system time")
		}
		// 等待时钟追上
		time.Sleep(time.Duration(backwardMillis) * time.Millisecond)
		timestamp = uint64(time.Now().UnixMilli()) - epochMillis
	}

	// 同一毫秒内，序列号递增。
	if timestamp == g.lastTime {
		g.sequence = (g.sequence + 1) & sequenceMask
		// 序列号溢出，等待下一毫秒。
		if g.sequence == 0 {
			for timestamp <= g.lastTime {
				time.Sleep(time.Millisecond)
				timestamp = uint64(time.Now().UnixMilli()) - epochMillis
			}
		}
	} else {
		// 新的毫秒，序列号重置。
		g.sequence = 0
	}

	g.lastTime = timestamp

	return (timestamp&timestampMask)<<timestampShift |
		(shardVal&hashMask)<<shardValShift |
		g.sequence, nil
}

// SnowflakeExtractor 实现了 ShardValExtractor 接口。
// 用于从 Snowflake ID 中提取分片值，可以注入到分片策略中。
type SnowflakeExtractor struct{}

// NewExtractor 创建一个新的 Snowflake 提取器实例。
func NewExtractor() *SnowflakeExtractor {
	return &SnowflakeExtractor{}
}

// ExtractShardVal 实现 ShardValExtractor 接口，从 ID 中提取分片值。
func (e *SnowflakeExtractor) ExtractShardVal(id uint64) uint64 {
	return (id >> shardValShift) & hashMask
}

// ExtractShardVal 独立函数，从 ID 中提取分片值。
// 便捷函数，无需创建 Extractor 实例。
func ExtractShardVal(id uint64) uint64 {
	return (id >> shardValShift) & hashMask
}

package sharding

import (
	"encoding/binary"

	"github.com/cespare/xxhash/v2"
)

// Sharder 是分片器的接口，用于提供分片值。
type Sharder interface {
	ShardVal() (uint64, error)
}

var _ Sharder = (*SingleIDSharder)(nil)

// SingleIDSharder 基于单个 ID 的分片器。
// 直接使用 ID 本身作为分片值，避免不必要的哈希计算。
type SingleIDSharder struct {
	id uint64
}

func NewSingleIDSharder(id uint64) SingleIDSharder {
	return SingleIDSharder{
		id: id,
	}
}

// ShardVal 直接返回 ID 本身作为分片值。
// 如果 ID 本身已经是均匀分布的 ( 如雪花算法生成的 ID )，
// 则不需要额外的哈希计算，性能更好。
func (s SingleIDSharder) ShardVal() (uint64, error) {
	return s.id, nil
}

var _ Sharder = (*BizSharder)(nil)

// BizSharder 基于业务 ID 和业务 Key 的分片器。
// 使用哈希算法将业务 ID 和 Key 组合后生成分片值。
type BizSharder struct {
	bizID  uint64
	bizKey string
}

func NewBizSharder(bizID uint64, bizKey string) BizSharder {
	return BizSharder{
		bizID:  bizID,
		bizKey: bizKey,
	}
}

// ShardVal 将业务 ID 和 Key 组合后进行哈希计算。
// 使用二进制拼接而非字符串拼接，性能更优。
func (s BizSharder) ShardVal() (uint64, error) {
	h := xxhash.New()

	// 写入 bizID 的二进制表示。
	buf := make([]byte, 8) //nolint:mnd // 8 bytes is enough for uint64
	binary.BigEndian.PutUint64(buf, s.bizID)
	if _, err := h.Write(buf); err != nil {
		return 0, err
	}

	// 写入分隔符
	if _, err := h.WriteString(":"); err != nil {
		return 0, err
	}

	// 写入 bizKey
	if _, err := h.WriteString(s.bizKey); err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

var _ Sharder = (*StringSharder)(nil)

// StringSharder 基于字符串的分片器。
// 适用于基于 email、mobile 等字符串标识进行分片的场景。
// 使用哈希算法保证分片均匀分布。
//
// 典型使用场景：
// - 生成用户 ID 时，基于 email 计算 shardVal。
// - 基于手机号、用户名等稳定标识进行分片。
type StringSharder struct {
	key string
}

func NewStringSharder(key string) StringSharder {
	return StringSharder{
		key: key,
	}
}

// ShardVal 对字符串进行哈希计算得到分片值。
// 使用 xxhash 算法保证均匀分布和高性能。
func (s StringSharder) ShardVal() (uint64, error) {
	return xxhash.Sum64String(s.key), nil
}

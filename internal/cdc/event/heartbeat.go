package event

// HeartbeatEvent 是 Debezium 心跳事件。
// 用于监控 Debezium connector 的健康状态。
//
//nolint:tagliatelle // Debezium 标准字段 ts_ms。
type HeartbeatEvent struct {
	// TimestampMs 心跳时间戳 ( 毫秒 )。
	// 这个时间戳表示 Debezium connector 最后一次成功读取数据的时间。
	TimestampMs int64 `json:"ts_ms"`
}

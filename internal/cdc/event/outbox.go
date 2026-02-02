package event

// OutboxCDCEvent 是 outbox 事件。
//
//nolint:tagliatelle // Debezium 内容字段于数据库字段保持一致。
type OutboxCDCEvent struct {
	CDCEvent
	ID uint64 `json:"id"`

	AggregateID   uint64 `json:"aggregate_id"`
	AggregateType string `json:"aggregate_type"`

	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	Metadata  string `json:"metadata"`

	RetryCount int `json:"retry_count"`

	CreatedAt int64 `json:"created_at"`
}

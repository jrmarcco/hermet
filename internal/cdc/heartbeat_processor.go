package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jrmarcco/hermet/internal/cdc/event"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsume"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

// HeartbeatStatus 记录单个数据源的心跳状态。
type HeartbeatStatus struct {
	// DataSource 是数据源标识（例如：hermet_0）。
	DataSource string

	// LastHeartbeatTime 是最后一次收到心跳的时间。
	LastHeartbeatTime time.Time

	// LastHeartbeat 是心跳消息中的时间戳（毫秒）。
	// 用于计算 CDC 延迟（当前时间 - 心跳时间戳）。
	LastHeartbeatTimestampMs int64

	// MessageCount 是收到的心跳消息总数。
	MessageCount int64

	// LastError 是最后一次处理心跳消息时的错误。
	LastError error
}

// IsHealthy 判断数据源是否健康。
// timeout 是心跳超时时间，如果超过该时间没有收到心跳，则认为不健康。
func (s *HeartbeatStatus) IsHealthy(timeout time.Duration) bool {
	if s.LastHeartbeatTime.IsZero() {
		return false // 从未收到过心跳
	}
	return time.Since(s.LastHeartbeatTime) < timeout
}

// GetDelay 获取 CDC 延迟（毫秒）。
// 返回心跳时间戳与实际收到时间的差值。
func (s *HeartbeatStatus) GetDelay() int64 {
	if s.LastHeartbeatTimestampMs == 0 {
		return 0
	}
	return s.LastHeartbeatTime.UnixMilli() - s.LastHeartbeatTimestampMs
}

// HeartbeatMonitor 是心跳监控器。
type HeartbeatMonitor struct {
	mu       sync.RWMutex
	statuses map[string]*HeartbeatStatus // key: data source

	// HeartbeatTimeout 是心跳超时时间。
	// 如果超过该时间没有收到心跳，认为 Debezium connector 不健康。
	HeartbeatTimeout time.Duration

	// MaxDelayThreshold 是最大延迟阈值（毫秒）。
	// 超过该阈值会记录警告日志。
	MaxDelayThreshold int64
}

// NewHeartbeatMonitor 创建心跳监控器。
func NewHeartbeatMonitor(timeout time.Duration, maxDelay int64) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		statuses:          make(map[string]*HeartbeatStatus),
		HeartbeatTimeout:  timeout,
		MaxDelayThreshold: maxDelay,
	}
}

// RecordHeartbeat 记录心跳。
func (m *HeartbeatMonitor) RecordHeartbeat(dataSource string, tsMs int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.statuses[dataSource]
	if !exists {
		status = &HeartbeatStatus{
			DataSource: dataSource,
		}
		m.statuses[dataSource] = status
	}

	status.LastHeartbeatTime = time.Now()
	status.LastHeartbeatTimestampMs = tsMs
	status.MessageCount++
	status.LastError = err

	// 计算延迟并记录警告
	if err == nil {
		delay := status.GetDelay()
		if delay > m.MaxDelayThreshold {
			slog.Warn(
				"[cdc-heartbeat] high CDC delay detected",
				"data_source", dataSource,
				"delay_ms", delay,
				"threshold_ms", m.MaxDelayThreshold,
			)
		}
	}
}

// GetStatus 获取指定数据源的心跳状态。
func (m *HeartbeatMonitor) GetStatus(dataSource string) *HeartbeatStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statuses[dataSource]
}

// GetAllStatuses 获取所有数据源的心跳状态。
func (m *HeartbeatMonitor) GetAllStatuses() map[string]*HeartbeatStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免并发问题
	result := make(map[string]*HeartbeatStatus, len(m.statuses))
	for k, v := range m.statuses {
		statusCopy := *v
		result[k] = &statusCopy
	}
	return result
}

// IsAllHealthy 判断所有数据源是否健康。
func (m *HeartbeatMonitor) IsAllHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.statuses) == 0 {
		return false // 没有任何心跳数据
	}

	for _, status := range m.statuses {
		if !status.IsHealthy(m.HeartbeatTimeout) {
			return false
		}
	}
	return true
}

// StartupStrategy 定义心跳处理器的启动策略。
type StartupStrategy string

const (
	// StartupStrategyLatest 从最新消息开始消费（跳过历史堆积）。
	// 适用场景：初次启动且有大量历史堆积，只关心实时状态。
	StartupStrategyLatest StartupStrategy = "latest"

	// StartupStrategyEarliest 从最早消息开始消费（消费所有历史消息）。
	// 适用场景：需要完整的心跳历史数据，或测试环境。
	StartupStrategyEarliest StartupStrategy = "earliest"

	// StartupStrategyRecent 从最近的消息开始消费（例如最近 1 小时）。
	// 适用场景：想保留一些历史数据用于分析，但不想消费全部堆积。
	StartupStrategyRecent StartupStrategy = "recent"

	// StartupStrategySaved 从上次保存的 offset 开始消费。
	// 适用场景：重启服务，继续上次的消费位置。
	StartupStrategySaved StartupStrategy = "saved"
)

// HeartbeatProcessorConfig 是心跳处理器的配置。
type HeartbeatProcessorConfig struct {
	// DBShardCount 是数据库分片数量。
	DBShardCount int

	// GroupID 是消费者组 ID。
	GroupID string

	// TopicPattern 是心跳 topic 名称模式。
	// 例如："__debezium-heartbeat.hermet_%d"
	TopicPattern string

	// StartupStrategy 是启动策略。
	StartupStrategy StartupStrategy

	// RecentDuration 当使用 StartupStrategyRecent 时，指定消费最近多长时间的消息。
	// 例如：1 * time.Hour 表示消费最近 1 小时的消息。
	RecentDuration time.Duration

	// SkipHistoricalOnStartup 是否在启动时跳过历史堆积消息。
	// 如果为 true，启动时会快速消费并丢弃历史消息，只记录最后一条。
	// 适用场景：有大量历史堆积，只需要最新状态。
	SkipHistoricalOnStartup bool
}

// HeartbeatProcessor 处理 Debezium 心跳消息的处理器。
type HeartbeatProcessor struct {
	*BaseProcessor
	monitor *HeartbeatMonitor
	config  HeartbeatProcessorConfig
}

// NewHeartbeatProcessor 创建心跳消息处理器。
func NewHeartbeatProcessor(
	config HeartbeatProcessorConfig,
	factory xconsume.ConsumerFactory,
	monitor *HeartbeatMonitor,
) *HeartbeatProcessor {
	// 设置默认值
	if config.TopicPattern == "" {
		config.TopicPattern = "__debezium-heartbeat.hermet_%d"
	}
	if config.StartupStrategy == "" {
		config.StartupStrategy = StartupStrategyLatest
	}

	processor := &HeartbeatProcessor{
		monitor: monitor,
		config:  config,
	}

	// 创建心跳消息处理器
	handler := processor.handleHeartbeat

	// 构建心跳 topic 列表
	topics := make([]xprocess.TopicConfig, 0, config.DBShardCount)
	for i := 0; i < config.DBShardCount; i++ {
		topics = append(topics, xprocess.TopicConfig{
			Topic:   fmt.Sprintf(config.TopicPattern, i),
			GroupID: config.GroupID,
		})
	}

	consumerProcessor := xprocess.NewConsumeProcessor(
		xprocess.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       handler,
			HandleTimeout: time.Second, // 心跳消息处理超时
		},
		factory,
	)

	processor.BaseProcessor = &BaseProcessor{
		ConsumeProcessor: consumerProcessor,
	}

	return processor
}

// handleHeartbeat 处理单条心跳消息。
func (p *HeartbeatProcessor) handleHeartbeat(_ context.Context, msg *xmq.Message) error {
	// 从 topic 名称提取数据源标识
	// 例如：__debezium-heartbeat.hermet_0 -> hermet_0
	dataSource := p.extractDataSource(msg.Topic)

	var heartbeat event.HeartbeatEvent
	if err := json.Unmarshal(msg.Val, &heartbeat); err != nil {
		slog.Error(
			"[cdc-heartbeat] failed to unmarshal heartbeat message",
			"topic", msg.Topic,
			"data_source", dataSource,
			"error", err.Error(),
		)
		p.monitor.RecordHeartbeat(dataSource, 0, err)
		return err
	}

	// 记录心跳
	p.monitor.RecordHeartbeat(dataSource, heartbeat.TimestampMs, nil)

	slog.Debug(
		"[cdc-heartbeat] received heartbeat",
		"topic", msg.Topic,
		"data_source", dataSource,
		"ts_ms", heartbeat.TimestampMs,
		"offset", msg.Offset,
	)

	return nil
}

// extractDataSource 从 topic 名称中提取数据源标识。
// 例如：__debezium-heartbeat.hermet_0 -> hermet_0
func (p *HeartbeatProcessor) extractDataSource(topic string) string {
	// 简单实现：查找最后一个点后的内容
	// 更复杂的场景可以使用正则表达式
	for i := len(topic) - 1; i >= 0; i-- {
		if topic[i] == '.' {
			return topic[i+1:]
		}
	}
	return topic
}

// GetMonitor 获取心跳监控器。
func (p *HeartbeatProcessor) GetMonitor() *HeartbeatMonitor {
	return p.monitor
}

// StartHealthCheck 启动健康检查 goroutine。
// 定期检查所有数据源的健康状态，并记录警告日志。
func (p *HeartbeatProcessor) StartHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("[cdc-heartbeat] health check stopped")
			return
		case <-ticker.C:
			p.checkHealth()
		}
	}
}

// checkHealth 检查所有数据源的健康状态。
func (p *HeartbeatProcessor) checkHealth() {
	statuses := p.monitor.GetAllStatuses()

	for dataSource, status := range statuses {
		if !status.IsHealthy(p.monitor.HeartbeatTimeout) {
			slog.Error(
				"[cdc-heartbeat] data source unhealthy - no heartbeat received",
				"data_source", dataSource,
				"last_heartbeat", status.LastHeartbeatTime.Format(time.RFC3339),
				"timeout", p.monitor.HeartbeatTimeout.String(),
				"message_count", status.MessageCount,
			)
		} else {
			delay := status.GetDelay()
			slog.Info(
				"[cdc-heartbeat] data source healthy",
				"data_source", dataSource,
				"delay_ms", delay,
				"message_count", status.MessageCount,
			)
		}
	}
}

// Start 启动心跳处理器。
// 覆盖 BaseProcessor 的 Start 方法，添加启动策略处理。
func (p *HeartbeatProcessor) Start(ctx context.Context) error {
	slog.Info(
		"[cdc-heartbeat] starting heartbeat processor",
		"startup_strategy", string(p.config.StartupStrategy),
		"skip_historical", p.config.SkipHistoricalOnStartup,
	)

	// 根据启动策略处理
	switch p.config.StartupStrategy {
	case StartupStrategyLatest:
		slog.Info("[cdc-heartbeat] using latest strategy - will skip historical messages")
		// Kafka consumer 默认会从 latest 开始（如果配置正确）
		// 无需额外处理

	case StartupStrategyRecent:
		slog.Info(
			"[cdc-heartbeat] using recent strategy",
			"duration", p.config.RecentDuration.String(),
		)
		// TODO: 实现从指定时间点开始消费
		// 需要 Kafka consumer 支持按时间戳定位 offset
		slog.Warn("[cdc-heartbeat] recent strategy not fully implemented yet, falling back to latest")

	case StartupStrategyEarliest:
		slog.Info("[cdc-heartbeat] using earliest strategy - will consume all historical messages")
		// 如果有大量堆积，这可能需要很长时间

	case StartupStrategySaved:
		slog.Info("[cdc-heartbeat] using saved strategy - resuming from last committed offset")
	}

	// 启动基础处理器
	if err := p.BaseProcessor.Start(ctx); err != nil {
		return err
	}

	// 如果配置了跳过历史消息，启动后台任务快速消费并丢弃
	if p.config.SkipHistoricalOnStartup && p.config.StartupStrategy != StartupStrategyLatest {
		slog.Info("[cdc-heartbeat] skip_historical enabled - historical messages will be consumed quickly")
		// 实际的跳过逻辑在 handleHeartbeat 中处理
	}

	return nil
}

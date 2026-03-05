package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jrmarcco/hermet/internal/cdc/event"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xconsumer"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
	"go.uber.org/zap"
)

// HeartbeatStatus 心跳状态。
// 用于记录单个数据源的心跳状态。
type HeartbeatStatus struct {
	DataSource string // 数据源标识 ( 对应 Debezium Connector 的 name 配置 )

	LastHeartbeatTime        time.Time // 最后一次收到心跳的时间 ( 即消费心跳消息的时间 )
	LastHeartbeatTimestampMs int64     // 心跳消息中的毫秒值 ( 用于计算 CDC 延迟 )

	MessageCount int64

	LastError error
}

// IsHealthy 判断数据源是否健康。
func (s *HeartbeatStatus) IsHealthy(timeout time.Duration) bool {
	if s.LastHeartbeatTime.IsZero() {
		// 从未收到过心跳。
		return false
	}
	return time.Since(s.LastHeartbeatTime) < timeout
}

// GetDelay 获取 CDC 延迟 ( 毫秒 )。
// 即心跳事件戳和实际收到心跳的差值。
func (s *HeartbeatStatus) GetDelay() int64 {
	if s.LastHeartbeatTimestampMs == 0 {
		return 0
	}
	return s.LastHeartbeatTime.UnixMilli() - s.LastHeartbeatTimestampMs
}

// HeartbeatMonitor 心跳监控器。
type HeartbeatMonitor struct {
	mu sync.RWMutex

	statuses map[string]*HeartbeatStatus // data source -> heartbeat status

	HeartbeatTimeout time.Duration
	DelayThreshold   int64
}

func NewHeartbeatMonitor(timeout time.Duration, maxDelay int64) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		statuses:         make(map[string]*HeartbeatStatus),
		HeartbeatTimeout: timeout,
		DelayThreshold:   maxDelay,
	}
}

func (m *HeartbeatMonitor) Record(ds string, tsMs int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, ok := m.statuses[ds]
	if !ok {
		status = &HeartbeatStatus{DataSource: ds}
		m.statuses[ds] = status
	}

	status.LastHeartbeatTime = time.Now()
	status.LastHeartbeatTimestampMs = tsMs
	status.MessageCount++
	status.LastError = err

	if err == nil {
		delay := status.GetDelay()
		if delay > m.DelayThreshold {
			slog.Warn(
				"[cdc-heartbeat] high CDC delay detected",
				"data_source", ds,
				"delay_ms", delay,
				"threshold_ms", m.DelayThreshold,
			)
		}
	}
}

func (m *HeartbeatMonitor) GetStatus(ds string) *HeartbeatStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statuses[ds]
}

func (m *HeartbeatMonitor) GetAllStatuses() map[string]*HeartbeatStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本 ( 避免并发问题 )。
	res := make(map[string]*HeartbeatStatus, len(m.statuses))
	for ds, status := range m.statuses {
		statusCopy := *status
		res[ds] = &statusCopy
	}
	return res
}

func (m *HeartbeatMonitor) IsAllHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.statuses) == 0 {
		// 没有任何心跳数据。
		return false
	}

	for _, status := range m.statuses {
		if !status.IsHealthy(m.HeartbeatTimeout) {
			return false
		}
	}
	return true
}

type HeartbeatProcessorConfig struct {
	GroupID string

	DataSources  []string // 数据源列表 ( 对应 Debezium Connector 的 name 配置 )
	TopicPattern string   // 心跳消息的 kafka topic 名称模式 ( 例如 "__debezium-heartbeat.%s" )
}

// HeartbeatProcessor 处理 Debezium 心跳消息的处理器。
type HeartbeatProcessor struct {
	*BaseProcessor

	monitor *HeartbeatMonitor
	logger  *zap.Logger
}

func NewHeartbeatProcessor(
	config HeartbeatProcessorConfig,
	factory xconsumer.ConsumerFactory,
	monitor *HeartbeatMonitor,
	logger *zap.Logger,
) (*HeartbeatProcessor, error) {
	if len(config.DataSources) == 0 {
		return nil, fmt.Errorf("data sources are required")
	}

	if config.TopicPattern == "" {
		return nil, fmt.Errorf("topic pattern is required")
	}

	processor := &HeartbeatProcessor{
		monitor: monitor,
		logger:  logger,
	}

	topics := make([]xprocessor.TopicConfig, 0, len(config.DataSources))
	for _, ds := range config.DataSources {
		topics = append(topics, xprocessor.TopicConfig{
			Topic:   fmt.Sprintf(config.TopicPattern, ds),
			GroupID: config.GroupID,
		})
	}

	consumerProcessor := xprocessor.NewConsumeProcessor(
		xprocessor.ConsumeProcessorConfig{
			Topics:        topics,
			Handler:       processor.processHeartbeat,
			HandleTimeout: time.Second,
		},
		factory,
	)

	processor.BaseProcessor = &BaseProcessor{
		ConsumeProcessor: consumerProcessor,
	}

	return processor, nil
}

func (p *HeartbeatProcessor) processHeartbeat(_ context.Context, msg *xmq.Message) error {
	ds := p.extractDataSource(msg.Topic)

	var heartbeat event.HeartbeatEvent
	if err := json.Unmarshal(msg.Val, &heartbeat); err != nil {
		p.logger.Error(
			"[cdc-heartbeat] failed to unmarshal heartbeat message",
			zap.String("topic", msg.Topic),
			zap.String("data_source", ds),
			zap.Error(err),
		)
		p.monitor.Record(ds, 0, err)
		return err
	}

	// 记录心跳。
	p.monitor.Record(ds, heartbeat.TimestampMs, nil)

	p.logger.Debug(
		"[cdc-heartbeat] received heartbeat",
		zap.String("topic", msg.Topic),
		zap.String("data_source", ds),
		zap.Int64("ts_ms", heartbeat.TimestampMs),
	)
	return nil
}

// extractDataSource 从 topic 名称中提取数据源标识。
func (p *HeartbeatProcessor) extractDataSource(topic string) string {
	return strings.Split(topic, ".")[1]
}

func (p *HeartbeatProcessor) StartHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("[cdc-heartbeat] health check stopped")
			return
		case <-ticker.C:
			p.healthCheck()
		}
	}
}

func (p *HeartbeatProcessor) healthCheck() {
	statuses := p.monitor.GetAllStatuses()

	for ds, status := range statuses {
		if !status.IsHealthy(p.monitor.HeartbeatTimeout) {
			p.logger.Error(
				"[cdc-heartbeat] data source unhealthy - no heartbeat received",
				zap.String("data_source", ds),
				zap.Time("last_heartbeat", status.LastHeartbeatTime),
			)
			continue
		}

		delay := status.GetDelay()
		p.logger.Debug(
			"[cdc-heartbeat] data source healthy",
			zap.String("data_source", ds),
			zap.Int64("delay_ms", delay),
			zap.Int64("message_count", status.MessageCount),
		)
	}
}

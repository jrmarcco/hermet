package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/cdc"
)

// HeartbeatService 是心跳监控服务接口。
type HeartbeatService interface {
	// GetHealthStatus 获取所有数据源的健康状态。
	GetHealthStatus() map[string]*cdc.HeartbeatStatus

	// IsHealthy 判断所有数据源是否健康。
	IsHealthy() bool

	// GetDataSourceStatus 获取指定数据源的健康状态。
	GetDataSourceStatus(dataSource string) (*cdc.HeartbeatStatus, error)

	// GetHealthReport 获取健康报告（人类可读格式）。
	GetHealthReport() string
}

var _ HeartbeatService = (*DefaultHeartbeatService)(nil)

// DefaultHeartbeatService 是默认的心跳监控服务实现。
type DefaultHeartbeatService struct {
	monitor *cdc.HeartbeatMonitor
}

// NewDefaultHeartbeatService 创建默认的心跳监控服务。
func NewDefaultHeartbeatService(monitor *cdc.HeartbeatMonitor) *DefaultHeartbeatService {
	return &DefaultHeartbeatService{
		monitor: monitor,
	}
}

func (s *DefaultHeartbeatService) GetHealthStatus() map[string]*cdc.HeartbeatStatus {
	return s.monitor.GetAllStatuses()
}

func (s *DefaultHeartbeatService) IsHealthy() bool {
	return s.monitor.IsAllHealthy()
}

func (s *DefaultHeartbeatService) GetDataSourceStatus(dataSource string) (*cdc.HeartbeatStatus, error) {
	status := s.monitor.GetStatus(dataSource)
	if status == nil {
		return nil, fmt.Errorf("data source not found: %s", dataSource)
	}
	return status, nil
}

func (s *DefaultHeartbeatService) GetHealthReport() string {
	statuses := s.GetHealthStatus()
	if len(statuses) == 0 {
		return "No heartbeat data available"
	}

	report := "=== CDC Heartbeat Health Report ===\n\n"
	overallHealthy := s.IsHealthy()
	if overallHealthy {
		report += "Overall Status: ✓ HEALTHY\n\n"
	} else {
		report += "Overall Status: ✗ UNHEALTHY\n\n"
	}

	for dataSource, status := range statuses {
		healthy := status.IsHealthy(s.monitor.HeartbeatTimeout)
		delay := status.GetDelay()

		report += fmt.Sprintf("Data Source: %s\n", dataSource)
		if healthy {
			report += "  Status: ✓ Healthy\n"
		} else {
			report += "  Status: ✗ Unhealthy\n"
		}
		report += fmt.Sprintf("  Last Heartbeat: %s\n", status.LastHeartbeatTime.Format(time.RFC3339))
		report += fmt.Sprintf("  CDC Delay: %d ms\n", delay)
		report += fmt.Sprintf("  Message Count: %d\n", status.MessageCount)
		if status.LastError != nil {
			report += fmt.Sprintf("  Last Error: %v\n", status.LastError)
		}
		report += "\n"
	}

	return report
}

// HeartbeatAlertHandler 是心跳告警处理器。
// 可以集成到告警系统（如钉钉、企业微信、PagerDuty 等）。
type HeartbeatAlertHandler interface {
	// OnUnhealthy 当检测到数据源不健康时触发。
	OnUnhealthy(ctx context.Context, dataSource string, status *cdc.HeartbeatStatus) error

	// OnHighDelay 当检测到高延迟时触发。
	OnHighDelay(ctx context.Context, dataSource string, delay int64) error
}

// LogAlertHandler 是基于日志的告警处理器（示例实现）。
type LogAlertHandler struct{}

func (h *LogAlertHandler) OnUnhealthy(_ context.Context, dataSource string, status *cdc.HeartbeatStatus) error {
	// 这里可以集成实际的告警系统
	// 例如：发送钉钉机器人消息、企业微信消息、邮件等
	fmt.Printf("[ALERT] Data source %s is unhealthy! Last heartbeat: %s\n",
		dataSource, status.LastHeartbeatTime.Format(time.RFC3339))
	return nil
}

func (h *LogAlertHandler) OnHighDelay(_ context.Context, dataSource string, delay int64) error {
	fmt.Printf("[ALERT] Data source %s has high delay: %d ms\n", dataSource, delay)
	return nil
}

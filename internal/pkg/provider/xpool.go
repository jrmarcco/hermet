package provider

import (
	"fmt"
	"time"

	"github.com/jrmarcco/jit/xpool"
	"github.com/spf13/viper"
)

// newBlockTaskPool 创建一个阻塞任务池。
// 任务池用于异步调用网关 grpc 消息进行消息推送。
//
// 这里针对群节点配置一些建议：
// 1. 配置可以参考流量估算公式 ( 实战很好用 )：
//   - C = 实例入池 QPS * grpc p95 ( 秒 )
//   - coreG = 1.2 * C
//   - maxG = 2.0~3.0 * C
//   - initG = 0.4~0.6 * coreG
//   - queueSize = 4~10 * coreG ( 低延迟有限取小 )
//
// 2. 中等规模 IM 集群节点推荐配置：
//   - initG: 			 64
//   - coreG: 			 128~192
//   - maxG: 			 256~384 ( initG/coreG/maxG 一般按并发需求 QPS * 延迟估算再给突发冗余 )
//   - queueSize: 		 1024~4096 ( 注意 queueSize 过大会放大排队时延 )
//   - queueBacklogRate: 0.25~0.40 ( 推荐先 0.30 - 积压 0 容易短抖动时频繁扩容 )
//   - maxIdleTime: 	 15s~30s ( 推荐 20s )
//   - submitTimeout: 	 30ms~150ms ( 推荐先 80ms - IM 在线消息通常宁愿快失败/降级也不要积压发送 )
//   - taskExecTimeout:  500ms~800ms ( 推荐先 800ms )
//     注意这里是全局兜底的任务执行超时时间。
//     推荐做法是在调用 Submit 时使用 WithTaskExecTimeoutInContext 按照业务分级设置超时时间：
//     1.高优先级消息：80~150ms
//     2. 普通聊天消息：150~300ms
//     3. 低优先级通知：300~800ms
//   - errQueueSize: 	 1024
//   - errWorkerCnt: 	 2~4
//   - errHandleTimeout: 200ms~1s ( 推荐 500ms )
//
// 3. 线上调参阈值 ( 看这几个指标就够了 )
//
//   - waitingCnt/queueSize 持续 > 0.6：
//     先升 coreG/maxG 或扩实例。
//   - submitRetryCnt 增长很快：
//     说明提交侧已饱和，优先加容量或缩短 submitTimeout 做更强背压。
//   - errDropCnt > 0：
//     错误处理队列或 worker 不够，增加 errQueueSize/errWorkerCnt。
//   - runningCnt 长期贴近 maxG：
//     容量不足或下游 grpc 变慢。
func newBlockTaskPool() (*xpool.BlockTaskPool, error) {
	type config struct {
		InitGoroutines int32 `mapstructure:"init_goroutines"`
		CoreGoroutines int32 `mapstructure:"core_goroutines"`
		MaxGoroutines  int32 `mapstructure:"max_goroutines"`

		QueueSize        int32   `mapstructure:"queue_size"`
		QueueBacklogRate float64 `mapstructure:"queue_backlog_rate"`

		MaxIdleTime     time.Duration `mapstructure:"max_idle_time"`
		SubmitTimeout   time.Duration `mapstructure:"submit_timeout"`
		TaskExecTimeout time.Duration `mapstructure:"task_exec_timeout"`

		ErrQueueSize     int32         `mapstructure:"err_queue_size"`
		ErrWorkerCnt     int32         `mapstructure:"err_worker_cnt"`
		ErrHandleTimeout time.Duration `mapstructure:"err_handle_timeout"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("hermet.task_pool", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task pool config: %w", err)
	}

	return xpool.NewBlockTaskPool(
		cfg.InitGoroutines,
		cfg.QueueSize,

		xpool.WithCoreG(cfg.CoreGoroutines),
		xpool.WithMaxG(cfg.MaxGoroutines),

		xpool.WithQueueBacklogRate(cfg.QueueBacklogRate),

		xpool.WithMaxIdleTime(cfg.MaxIdleTime),
		xpool.WithSubmitTimeout(cfg.SubmitTimeout),
		xpool.WithTaskExecTimeout(cfg.TaskExecTimeout),

		xpool.WithErrQueueSize(cfg.ErrQueueSize),
		xpool.WithErrWorkerCnt(cfg.ErrWorkerCnt),
		xpool.WithErrHandleTimeout(cfg.ErrHandleTimeout),
	)
}

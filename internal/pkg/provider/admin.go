package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/pkg/buildinfo"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// readinessCacheTTL 为就绪检查结果的缓存时长，
// 防止探针高频打爆依赖 ( 结果最多延迟 5s )。
const readinessCacheTTL = 5 * time.Second

// 就绪/存活状态值。
const (
	statusOK       = "ok"
	statusDegraded = "degraded"
)

// AdminFxModule 提供运维管理端口:
//
//	GET /healthz          存活探针 ( 进程可用 )
//	GET /readyz           就绪探针 ( 依赖健康，结果缓存 5s )
//	GET /version          构建版本信息
//	GET /metrics          Prometheus 指标
//	GET /debug/pprof/*    Go pprof
//
// 管理端口与业务端口隔离，避免 pprof 与指标抓取影响数据面。
// 未配置 hermet.admin.addr 时不启用。
var AdminFxModule = fx.Module(
	"admin",
	fx.Invoke(startAdminServer),
)

type adminConfig struct {
	Addr            string        `mapstructure:"addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

func startAdminServer(
	rdb redis.Cmdable,
	esClient *elasticsearch.Client,
	lifecycle fx.Lifecycle,
) error {
	cfg := adminConfig{}
	if err := viper.UnmarshalKey("hermet.admin", &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal admin config: %w", err)
	}

	if cfg.Addr == "" {
		// 未配置则不启用管理端口。
		return nil
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/readyz", newReadinessHandler(rdb, esClient))
	mux.HandleFunc("/version", handleVersion)
	mux.Handle("/metrics", promhttp.Handler())

	// pprof 端点 ( 标准库默认路径 )。
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, //nolint:mnd // 防慢请求占用连接
	}

	logger := logging.For("admin")

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lc := net.ListenConfig{}
			ln, err := lc.Listen(ctx, "tcp", cfg.Addr)
			if err != nil {
				return fmt.Errorf("failed to listen admin server: %w", err)
			}

			go func() {
				if serveErr := server.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
					logger.Error("admin server exited with error", "error", serveErr)
				}
			}()

			logger.Info("admin server started", "addr", cfg.Addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Warn("failed to shutdown admin server", "error", err)
			}
			return nil
		},
	})

	return nil
}

// handleHealthz 存活探针: 进程与 admin 端口可用即返回 200。
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": statusOK})
}

// handleVersion 返回构建版本信息。
func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": buildinfo.Version,
		"commit":  buildinfo.Commit,
		"date":    buildinfo.Date,
	})
}

// newReadinessHandler 创建就绪探针，结果缓存 readinessCacheTTL。
//
// 状态判定:
//   - redis 不可用 → 整体 degraded ( session/鉴权依赖，不可降级 )；
//   - es 不可用 → 仅展示分项状态，不影响整体结论
//     ( ES 是查询加速层，挂掉只影响搜索，消息主链路不受影响，可后置恢复 )。
func newReadinessHandler(rdb redis.Cmdable, esClient *elasticsearch.Client) http.HandlerFunc {
	var (
		mu        sync.Mutex
		cachedAt  time.Time
		cachedRes readinessResult
	)

	check := func() readinessResult {
		res := readinessResult{
			Status:  statusOK,
			Checks:  map[string]string{"redis": statusOK, "es": statusOK},
			Checked: time.Now().Format(time.RFC3339),
		}

		// Redis: 1s 超时 PING。
		if rdb == nil {
			res.Status = statusDegraded
			res.Checks["redis"] = "not configured"
		} else {
			pingCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := rdb.Ping(pingCtx).Err(); err != nil {
				res.Status = statusDegraded
				res.Checks["redis"] = err.Error()
			}
		}

		// ES: 信息展示，不影响整体状态 ( 加速层可降级 )。
		if esClient == nil {
			res.Checks["es"] = "not configured"
		} else {
			pingCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resp, err := esClient.Info(esClient.Info.WithContext(pingCtx))
			if err != nil {
				res.Checks["es"] = err.Error()
			} else {
				_ = resp.Body.Close()
			}
		}

		return res
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		cached := !cachedAt.IsZero() && time.Since(cachedAt) < readinessCacheTTL
		result := cachedRes
		mu.Unlock()

		if !cached {
			result = check()
			mu.Lock()
			cachedRes, cachedAt = result, time.Now()
			mu.Unlock()
		}

		status := http.StatusOK
		if result.Status != statusOK {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, result)
	}
}

type readinessResult struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks"`
	Checked string            `json:"checked"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

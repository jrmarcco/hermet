package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AdminFxModule 提供运维管理端口 ( /metrics + /debug/pprof )。
// 与业务端口隔离，避免 pprof/metrics 暴露到公网入口。
var AdminFxModule = fx.Module(
	"admin",
	fx.Invoke(startAdminServer),
)

type adminConfig struct {
	Addr            string        `mapstructure:"addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

func startAdminServer(logger *zap.Logger, lifecycle fx.Lifecycle) error {
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

	mux.Handle("/metrics", promhttp.Handler())

	// pprof 标准路径。
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	svr := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, //nolint:mnd // 防慢请求占用连接
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("[admin] admin server exited with error", zap.Error(err))
				}
			}()

			logger.Info("[admin] admin server started", zap.String("addr", cfg.Addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
			defer cancel()

			return svr.Shutdown(stopCtx)
		},
	})

	return nil
}

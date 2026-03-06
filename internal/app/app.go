package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var AppFxModule = fx.Module(
	"app",
	fx.Provide(
		gin.Default,
	),
	fx.Invoke(InitApp),
)

type App struct {
	svr    *http.Server
	logger *zap.Logger
}

func (app *App) Start() error {
	go func() {
		if err := app.svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Fatal("[app] failed to start http server", zap.Error(err))
		}
	}()

	app.logger.Info("[app] http server started")
	return nil
}

func (app *App) Stop(ctx context.Context) error {
	if err := app.svr.Shutdown(ctx); err != nil {
		app.logger.Error("[app] failed to shutdown http server", zap.Error(err))
		return err
	}
	return nil
}

// appFxParams 是 app 的依赖注入参数。
type appFxParams struct {
	fx.In

	Builders   []xgin.HandlerFuncBuilder `group:"api_middleware"`
	Registries []xgin.RouteRegistry      `group:"api_registry"`

	Engine *gin.Engine
	Logger *zap.Logger

	Lifecycle fx.Lifecycle
}

type webConfig struct {
	Addr              string        `mapstructure:"addr"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"` // 读取请求头超时，防止 Slowloris 攻击
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`        // 读取整个请求超时，IM 请求通常较小
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`       // 写响应超时，历史记录数据量可能较大
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`        // Keep-Alive 空闲超时，IM 客户端会频繁请求
}

func InitApp(params appFxParams) (*App, error) {
	cfg, err := loadWebConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load web config: %w", err)
	}

	// 注册 Middleware ( AOP )。
	if len(params.Builders) > 0 {
		middlewares := make([]gin.HandlerFunc, 0, len(params.Builders))
		for _, builder := range params.Builders {
			middlewares = append(middlewares, builder.Build())
		}
		params.Engine.Use(middlewares...)
	}

	svr := &http.Server{
		Addr:              cfg.Addr,
		Handler:           params.Engine.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	app := &App{
		svr:    svr,
		logger: params.Logger,
	}

	if len(params.Registries) > 0 {
		for _, registry := range params.Registries {
			registry.Register(params.Engine)
		}
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return app.Start()
		},
		OnStop: func(ctx context.Context) error {
			return app.Stop(ctx)
		},
	})

	return app, nil
}

func loadWebConfig() (*webConfig, error) {
	cfg := &webConfig{}
	if err := viper.UnmarshalKey("hermet.web", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal web config: %w", err)
	}
	return cfg, nil
}

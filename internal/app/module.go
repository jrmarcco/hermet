package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/xgrpc/register"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var AppFxModule = fx.Module(
	"app",
	fx.Provide(
		gin.Default,
	),
	fx.Invoke(newApp),
)

// appFxParams 是 app 的依赖注入参数。
type appFxParams struct {
	fx.In

	Builders   []xgin.HandlerFuncBuilder `group:"api_middleware"`
	Registries []xgin.RouteRegistry      `group:"api_registry"`

	Engine *gin.Engine

	GrpcServer   *grpc.Server
	GrpcRegistry register.Registry
	GrpcInstance register.ServiceInstance

	Logger *zap.Logger

	Lifecycle fx.Lifecycle
}

type webConfig struct {
	Addr string `mapstructure:"addr"`

	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"` // 读取请求头超时，防止 Slowloris 攻击
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`        // 读取整个请求超时，IM 请求通常较小
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`       // 写响应超时，历史记录数据量可能较大
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`        // Keep-Alive 空闲超时，IM 客户端会频繁请求
}

type grpcConfig struct {
	Addr string `mapstructure:"addr"`
}

func newApp(params appFxParams) (*App, error) {
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

	grpcCfg, err := loadGrpcAddrConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load grpc config: %w", err)
	}

	app := &App{
		svr:    svr,
		logger: params.Logger,

		grpcServer:   params.GrpcServer,
		grpcAddr:     grpcCfg.Addr,
		grpcRegistry: params.GrpcRegistry,
		grpcInstance: params.GrpcInstance,
	}

	if len(params.Registries) > 0 {
		for _, registry := range params.Registries {
			registry.Register(params.Engine)
		}
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return app.Start(ctx)
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

func loadGrpcAddrConfig() (*grpcConfig, error) {
	cfg := &grpcConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.server", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc server config: %w", err)
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("hermet.grpc.server.addr is required")
	}
	return cfg, nil
}

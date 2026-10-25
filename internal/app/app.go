package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/jrmarcco/hermet/internal/pkg/logging"

	"github.com/jrmarcco/xgrpc/register"
	"google.golang.org/grpc"
)

type App struct {
	svr *http.Server

	grpcServer   *grpc.Server
	grpcAddr     string
	grpcRegistry register.Registry
	grpcInstance register.ServiceInstance
}

// Start 启动 gRPC 服务 ( 含 etcd 注册 ) 与 HTTP 服务。
//
// 启动顺序: 先 gRPC 后 HTTP——保证 HTTP 就绪时网关已经可以发现并调用 SyncService。
func (app *App) Start(ctx context.Context) error {
	if err := app.startGrpc(ctx); err != nil {
		return err
	}

	go func() {
		if err := app.svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.For("app").Error("failed to start http server", "error", err)
		}
	}()

	logging.For("app").Info("http server started")
	return nil
}

// Stop 停止 HTTP 与 gRPC 服务 ( 注销 etcd 注册，优雅排空存量连接 )。
func (app *App) Stop(ctx context.Context) error {
	var errs []error

	if err := app.svr.Shutdown(ctx); err != nil {
		logging.For("app").Error("failed to shutdown http server", "error", err)
		errs = append(errs, fmt.Errorf("[app] failed to shutdown http server: %w", err))
	}

	if err := app.unregisterGrpc(ctx); err != nil {
		errs = append(errs, fmt.Errorf("[app] failed to unregister grpc service: %w", err))
	}

	app.stopGrpc()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (app *App) startGrpc(ctx context.Context) error {
	port, err := grpcListenPortFromAddr(app.grpcAddr)
	if err != nil {
		return err
	}

	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", fmt.Sprintf("[::]:%s", port))
	if err != nil {
		return fmt.Errorf("[app] failed to listen grpc: %w", err)
	}

	go func() {
		if serveErr := app.grpcServer.Serve(ln); serveErr != nil && !isExpectedGRPCServeErr(serveErr) {
			logging.For("app").Error("grpc serve exited with error", "error", serveErr)
		}
	}()

	if app.grpcRegistry != nil && app.grpcInstance.Name != "" {
		if err := app.grpcRegistry.Register(ctx, app.grpcInstance); err != nil {
			return fmt.Errorf("[app] failed to register grpc service: %w", err)
		}
		logging.For("app").Info("grpc service registered", "instance", app.grpcInstance.Addr)
	}

	logging.For("app").Info("grpc server started", "addr", app.grpcAddr)
	return nil
}

func (app *App) unregisterGrpc(ctx context.Context) error {
	if app.grpcRegistry == nil || app.grpcInstance.Name == "" {
		return nil
	}
	return app.grpcRegistry.Unregister(ctx, app.grpcInstance)
}

func (app *App) stopGrpc() {
	if app.grpcServer == nil {
		return
	}
	app.grpcServer.GracefulStop()
}

func grpcListenPortFromAddr(addr string) (string, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("[app] invalid grpc service address %q: %w", addr, err)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", fmt.Errorf("[app] invalid grpc service port %q: %w", port, err)
	}
	return port, nil
}

func isExpectedGRPCServeErr(err error) bool {
	if errors.Is(err, grpc.ErrServerStopped) {
		return true
	}
	// grpc 服务停止时，底层 listener 关闭会返回这一类错误。
	return strings.Contains(err.Error(), "use of closed network connection")
}

package grpc

import (
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

// GrpcFxModule 提供 gRPC Server 并注册 SyncService。
var GrpcFxModule = fx.Module(
	"grpc",
	fx.Provide(
		NewSyncServer,
		newGrpcServer,
	),
)

func newGrpcServer(syncServer *SyncServer) *grpc.Server {
	svr := grpc.NewServer()
	messagev1.RegisterSyncServiceServer(svr, syncServer)
	return svr
}

package provider

import (
	"context"
	"fmt"
	"time"

	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/jrmarcco/xgrpc/client/bp"
	"github.com/jrmarcco/xgrpc/client/rr"
	"github.com/jrmarcco/xgrpc/register/etcd"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// NotifyClientFxModule 提供网关 NotifyService 的 gRPC 客户端
// ( Kafka 通知链路之外的即时通道，契约见 proto/message/v1/notify.proto )。
//
// 语义: 网关节点只通知本节点在线的连接，未命中由 Kafka 广播链路兜底——
// 因此这里用轮询负载均衡 ( 任意节点命中即低延迟，未命中可接受 )，
// 调用失败无需重试 ( 通知只影响实时性 )。
var NotifyClientFxModule = fx.Module(
	"notify-client",
	fx.Provide(newNotifyServiceClientManager),
)

func newNotifyServiceClientManager(
	etcdClient *clientv3.Client,
	lifecycle fx.Lifecycle,
) (*client.Manager[messagev1.NotifyServiceClient], error) {
	cfg := grpcClientConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.client.notify", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notify client config: %w", err)
	}

	registry, err := etcd.NewBuilder(etcdClient).
		KeyPrefix(cfg.KeyPrefix).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd registry: %w", err)
	}

	rb, err := rr.NewResolverBuilder(registry, cfg.ResolverTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver builder: %w", err)
	}

	// 轮询: 通知是尽力而为的即时通道，无需一致性哈希。
	manager := client.NewManagerBuilder(
		rb,
		bp.NewRoundRobin(),
		func(conn *grpc.ClientConn) messagev1.NotifyServiceClient {
			return messagev1.NewNotifyServiceClient(conn)
		},
	).KeepAlive(keepalive.ClientParameters{
		Time:                cfg.KeepaliveTime,
		Timeout:             cfg.KeepaliveTimeout,
		PermitWithoutStream: true,
	}).Insecure().Build()

	lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return manager.CloseAll()
		},
	})

	return manager, nil
}

type grpcClientConfig struct {
	KeyPrefix        string        `mapstructure:"key_prefix"`
	ResolverTimeout  time.Duration `mapstructure:"resolver_timeout"`
	KeepaliveTime    time.Duration `mapstructure:"keepalive_time"`
	KeepaliveTimeout time.Duration `mapstructure:"keepalive_timeout"`
}

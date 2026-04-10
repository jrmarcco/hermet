package provider

import (
	"fmt"
	"time"

	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	nodev1 "github.com/jrmarcco/synp-api/api/go/node/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/jrmarcco/xgrpc/client/bp"
	"github.com/jrmarcco/xgrpc/client/rr"
	"github.com/jrmarcco/xgrpc/register/etcd"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/keepalive"
)

type grpcClientConfig struct {
	KeyPrefix        string        `mapstructure:"key_prefix"`
	ResolverTimeout  time.Duration `mapstructure:"resolver_timeout"`
	KeepaliveTime    time.Duration `mapstructure:"keepalive_time"`
	KeepaliveTimeout time.Duration `mapstructure:"keepalive_timeout"`
}

func newGrpcPushClient(etcdClient *clientv3.Client) (*client.Manager[messagev1.PushServiceClient], error) {
	cfg := grpcClientConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.client.message_push", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc client config: %w", err)
	}

	return createGrpcClientManager(etcdClient, cfg, bp.NewRoundRobin(), func(conn *grpc.ClientConn) messagev1.PushServiceClient {
		return messagev1.NewPushServiceClient(conn)
	})
}

func newGrpcLocateClient(etcdClient *clientv3.Client) (*client.Manager[nodev1.LocateServiceClient], error) {
	cfg := grpcClientConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.client.node_locate", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc client config: %w", err)
	}

	// 这里需要使用一致哈希负载均衡。
	// 因为网关使用本地缓存保存节点定位信息，一致性哈希可以保证缓存命中。
	bb, err := bp.NewConsistentHash()
	if err != nil {
		return nil, fmt.Errorf("failed to create consistent hash balancer: %w", err)
	}
	return createGrpcClientManager(etcdClient, cfg, bb, func(conn *grpc.ClientConn) nodev1.LocateServiceClient {
		return nodev1.NewLocateServiceClient(conn)
	})
}

func createGrpcClientManager[T any](
	etcdClient *clientv3.Client,
	cfg grpcClientConfig,
	bb balancer.Builder,
	factory func(conn *grpc.ClientConn) T,
) (*client.Manager[T], error) {
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

	manager := client.NewManagerBuilder(
		rb,
		bb,
		factory,
	).KeepAlive(keepalive.ClientParameters{
		Time:                cfg.KeepaliveTime,
		Timeout:             cfg.KeepaliveTimeout,
		PermitWithoutStream: true,
	}).Insecure().Build()
	return manager, nil
}

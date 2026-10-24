package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/xgrpc/register"
	"github.com/jrmarcco/xgrpc/register/etcd"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
)

// GrpcRegistryFxModule 提供 gRPC 服务注册器。
// backend 以 synp-backend 服务名注册，网关经 synp:///synp-backend 服务发现调用 SyncService。
var GrpcRegistryFxModule = fx.Module(
	"grpc-registry",
	fx.Provide(newGrpcRegistry),
)

func newGrpcRegistry(etcdClient *clientv3.Client, lifecycle fx.Lifecycle) (register.Registry, error) {
	cfg, err := loadGrpcRegistryConfig()
	if err != nil {
		return nil, err
	}

	registry, err := etcd.NewBuilder(etcdClient).
		KeyPrefix(cfg.KeyPrefix).
		LeaseTTL(cfg.LeaseTTL).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc registry: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return registry.Close()
		},
	})

	return registry, nil
}

// GrpcServiceInstanceFxModule 提供 gRPC 服务实例信息。
var GrpcServiceInstanceFxModule = fx.Module(
	"grpc-service-instance",
	fx.Provide(newGrpcServiceInstance),
)

func newGrpcServiceInstance() (register.ServiceInstance, error) {
	cfg := grpcServerConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.server", &cfg); err != nil {
		return register.ServiceInstance{}, fmt.Errorf("failed to unmarshal grpc server config: %w", err)
	}

	if cfg.Addr == "" {
		return register.ServiceInstance{}, fmt.Errorf("hermet.grpc.server.addr is required")
	}

	return register.ServiceInstance{
		Name:  cfg.Name,
		Addr:  cfg.Addr,
		Group: cfg.Group,
	}, nil
}

type grpcServerConfig struct {
	Name string `mapstructure:"name"`
	Addr string `mapstructure:"addr"`

	Group string `mapstructure:"group"`

	Registry grpcRegistryConfig `mapstructure:"registry"`
}

type grpcRegistryConfig struct {
	KeyPrefix string        `mapstructure:"key_prefix"`
	LeaseTTL  int           `mapstructure:"lease_ttl"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

func loadGrpcRegistryConfig() (*grpcRegistryConfig, error) {
	cfg := &grpcServerConfig{}
	if err := viper.UnmarshalKey("hermet.grpc.server", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc server config: %w", err)
	}
	return &cfg.Registry, nil
}

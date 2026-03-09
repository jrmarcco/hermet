package provider

import (
	"fmt"
	"time"

	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/jrmarcco/xgrpc/client/bp"
	"github.com/jrmarcco/xgrpc/client/rr"
	"github.com/jrmarcco/xgrpc/register/etcd"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func newGrpcPushClient(etcdClient *clientv3.Client) (*client.Manager[messagev1.PushServiceClient], error) {
	type config struct {
		KeyPrefix        string        `mapstructure:"key_prefix"`
		ResolverTimeout  time.Duration `mapstructure:"resolver_timeout"`
		KeepaliveTime    time.Duration `mapstructure:"keepalive_time"`
		KeepaliveTimeout time.Duration `mapstructure:"keepalive_timeout"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("hermet.grpc.client.message_push", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc config: %w", err)
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

	manager := client.NewManagerBuilder(
		rb,
		bp.NewRoundRobin(),
		func(conn *grpc.ClientConn) messagev1.PushServiceClient {
			return messagev1.NewPushServiceClient(conn)
		},
	).KeepAlive(keepalive.ClientParameters{
		Time:                cfg.KeepaliveTime,
		Timeout:             cfg.KeepaliveTimeout,
		PermitWithoutStream: true,
	}).Insecure().Build()

	return manager, nil
}

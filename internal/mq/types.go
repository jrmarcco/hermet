package mq

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

// Producer 是业务消息生产者接口。
type Producer interface {
	Produce(ctx context.Context, key, val []byte) error
}

// Consumer 是业务消息消费者接口。
type Consumer interface {
	xprocess.Processor
}

// Processor 是业务消息处理器接口 ( 组合了 Producer 和 Consumer )。
type Processor interface {
	Producer
	Consumer
}

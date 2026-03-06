package mq

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
)

//go:generate mockgen -source=types.go -destination=mock/mq.mock.go -package=mqmock -typed Producer,Consumer

// Producer 是业务消息生产者接口。
type Producer interface {
	Produce(ctx context.Context, key, val []byte) error
}

// Consumer 是业务消息消费者接口。
type Consumer interface {
	xprocessor.Processor
}

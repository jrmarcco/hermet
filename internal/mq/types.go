package mq

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
)

// Processor 是业务消息处理器接口。
type Processor interface {
	xprocess.Processor
	Produce(ctx context.Context, key []byte, val []byte) error
}

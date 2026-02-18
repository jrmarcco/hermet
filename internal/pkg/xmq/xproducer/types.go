package xproducer

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
)

type Producer interface {
	Produce(ctx context.Context, msg *xmq.Message) error
}

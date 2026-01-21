package cdc

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
)

type ProcessFunc func(ctx context.Context, msg *xmq.Message) error

type Processor interface {
	Start(processFunc ProcessFunc) error
}

package cdc

import "github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"

// Processor 是 CDC 处理器接口。
type Processor interface {
	xprocessor.Processor
}

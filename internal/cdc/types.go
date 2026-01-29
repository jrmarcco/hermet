package cdc

import "github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"

// Processor 是 CDC 处理器接口。
type Processor interface {
	xprocess.Processor
}

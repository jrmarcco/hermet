package mq

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocessor"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproducer"
)

var _ Producer = (*DefaultProducer)(nil)

type DefaultProducer struct {
	producer *xprocessor.ProduceProcessor
}

func NewDefaultProducer(topic string, producer xproducer.Producer) (*DefaultProducer, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	pp, err := xprocessor.NewProduceProcessor(topic, producer)
	if err != nil {
		return nil, err
	}

	return &DefaultProducer{
		producer: pp,
	}, nil
}

func (p *DefaultProducer) Produce(ctx context.Context, key, val []byte) error {
	return p.producer.Produce(ctx, key, val)
}

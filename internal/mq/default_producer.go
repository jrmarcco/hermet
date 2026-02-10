package mq

import (
	"context"
	"fmt"

	"github.com/jrmarcco/hermet/internal/pkg/xmq/xprocess"
	"github.com/jrmarcco/hermet/internal/pkg/xmq/xproduce"
)

var _ Producer = (*DefaultProducer)(nil)

type DefaultProducer struct {
	producer *xprocess.ProduceProcessor
}

func NewDefaultProducer(topic string, producer xproduce.Producer) (*DefaultProducer, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	pp, err := xprocess.NewProduceProcessor(topic, producer)
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

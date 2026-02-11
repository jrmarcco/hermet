package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/mq"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, msg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	uMsgConsumer mq.Consumer
	dMsgProducer mq.Producer
}

func NewDefaultMessageService(uMsgConsumer mq.Consumer, dMsgProducer mq.Producer) *DefaultMessageService {
	return &DefaultMessageService{
		uMsgConsumer: uMsgConsumer,
		dMsgProducer: dMsgProducer,
	}
}

func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, _ *xmq.Message) error {
	// TODO: not implemented
	panic("not implemented")
}

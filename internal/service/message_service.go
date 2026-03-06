package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, msg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct{}

func NewDefaultMessageService() *DefaultMessageService {
	return &DefaultMessageService{}
}

func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, _ *xmq.Message) error {
	// TODO: not implemented
	panic("not implemented")
}

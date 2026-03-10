package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, msg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	messageRepo repo.MessageRepo
	clients     *client.Manager[messagev1.PushServiceClient]
}

func NewDefaultMessageService(
	messageRepo repo.MessageRepo,
	clients *client.Manager[messagev1.PushServiceClient],
) *DefaultMessageService {
	return &DefaultMessageService{
		messageRepo: messageRepo,
		clients:     clients,
	}
}

func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, _ *xmq.Message) error {
	// TODO: not implemented
	panic("not implemented")
}

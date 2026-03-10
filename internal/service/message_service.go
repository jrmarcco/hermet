package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
	"go.uber.org/zap"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, msg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	messageRepo repo.MessageRepo

	clients *client.Manager[messagev1.PushServiceClient]
	logger  *zap.Logger
}

func NewDefaultMessageService(
	messageRepo repo.MessageRepo,
	clients *client.Manager[messagev1.PushServiceClient],
	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		messageRepo: messageRepo,
		clients:     clients,
		logger:      logger,
	}
}

func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, msg *xmq.Message) error {
	// TODO: not implemented
	s.logger.Debug("processing upstream message", zap.Any("message", msg))
	return nil
}

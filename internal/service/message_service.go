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

// ProcessUpstreamMsg 处理上游消息。
//
// 这里需要做 2 件事：
//  1. 保存消息到数据库 ( 这里是 MongoDB )。
//  2. 调用网关 grpc 接口进行消息推送。
func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, _ *xmq.Message) error {
	// TODO: not implemented
	return nil
}

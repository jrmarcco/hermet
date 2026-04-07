package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, msg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	rdb redis.Cmdable

	messageRepo repo.MessageRepo

	pushServiceClients *client.Manager[messagev1.PushServiceClient]
	logger             *zap.Logger
}

func NewDefaultMessageService(
	rdb redis.Cmdable,
	messageRepo repo.MessageRepo,
	clients *client.Manager[messagev1.PushServiceClient],
	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		rdb:                rdb,
		messageRepo:        messageRepo,
		pushServiceClients: clients,
		logger:             logger,
	}
}

// ProcessUpstreamMsg 处理上游消息。
//
// 这里需要做 2 件事：
//  1. 保存消息到数据库 ( 这里是 MongoDB )。
//  2. 调用网关 grpc 接口推送消息。
func (s *DefaultMessageService) ProcessUpstreamMsg(_ context.Context, message *xmq.Message) error {
	// 保存消息到数据库。
	domainMsg := domain.Message{}
	if err := json.Unmarshal(message.Val, &domainMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// 调用网关 grpc 接口推送消息。
	// 这里要分情况处理 ( 单聊 / 群聊 )。
	if domainMsg.DestinationType == domain.ChannelTypeSingle {
		// 单聊：直接推送消息。
	}

	// 群聊：查询群成员列表，然后批量推送消息。
	return nil
}

// pushSingleMsg 推送单聊消息。
//
//nolint:unused // TODO: implement
func (s *DefaultMessageService) pushSingleMsg(_ context.Context, _ *domain.Message) error {
	// 1. 查询目标用户所在节点。
	return nil
}

// pushGroupMsg 推送群聊消息。
//
//nolint:unused // TODO: implement
func (s *DefaultMessageService) pushGroupMsg(_ context.Context, _ *domain.Message) error {
	return nil
}

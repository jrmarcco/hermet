package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xpool"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	nodev1 "github.com/jrmarcco/synp-api/api/go/node/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, upstreamMsg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	rdb           redis.Cmdable
	messageRepo   repo.MessageRepo
	blockTaskPool *xpool.BlockTaskPool

	pushServiceClients   *client.Manager[messagev1.PushServiceClient]
	locateServiceName    string
	locateServiceClients *client.Manager[nodev1.LocateServiceClient]
	logger               *zap.Logger
}

func NewDefaultMessageService(
	rdb redis.Cmdable,
	messageRepo repo.MessageRepo,
	blockTaskPool *xpool.BlockTaskPool,
	pushServiceClients *client.Manager[messagev1.PushServiceClient],
	locateServiceClients *client.Manager[nodev1.LocateServiceClient],
	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		rdb:           rdb,
		messageRepo:   messageRepo,
		blockTaskPool: blockTaskPool,

		pushServiceClients:   pushServiceClients,
		locateServiceClients: locateServiceClients,

		logger: logger,
	}
}

// ProcessUpstreamMsg 处理上游消息。
//
// 这里需要做 2 件事：
//  1. 保存消息到数据库 ( 这里是 MongoDB )。
//  2. 调用网关 grpc 接口推送消息。
func (s *DefaultMessageService) ProcessUpstreamMsg(ctx context.Context, upstreamMsg *xmq.Message) error {
	// 保存消息到数据库。
	domainMsg := domain.Message{}
	if err := json.Unmarshal(upstreamMsg.Val, &domainMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// 调用网关 grpc 接口推送消息。
	// 这里要分情况处理 ( 单聊 / 群聊 )。
	if domainMsg.DestinationType == domain.ChannelTypeSingle {
		// 单聊：直接推送消息。
		return s.pushSingleMsg(ctx, &domainMsg)
	}

	// 群聊：查询群成员列表，然后批量推送消息。
	return nil
}

// pushSingleMsg 推送单聊消息。
func (s *DefaultMessageService) pushSingleMsg(ctx context.Context, msg *domain.Message) error {
	// 1. 查询目标用户所在节点。
	addrs, err := s.locateUserAddr(ctx, msg.DestinationID)
	if err != nil {
		return fmt.Errorf("failed to push single message to user: %w", err)
	}

	for _, addr := range addrs {
		// TODO: 使用线程池进行消息推送。
		s.logger.Debug("pushing single message to user", zap.Uint64("user_id", msg.DestinationID), zap.String("addr", addr))
	}
	return nil
}

// pushGroupMsg 推送群聊消息。
//
//nolint:unused // TODO: implement
func (s *DefaultMessageService) pushGroupMsg(_ context.Context, _ *domain.Message) error {
	return nil
}

// locateUserAddr 定位用户所在的网关地址列表。
// 这里只需要地址，推送消息通过地址定向调用。
func (s *DefaultMessageService) locateUserAddr(ctx context.Context, uid uint64) ([]string, error) {
	grpcClient, err := s.locateServiceClients.Get(s.locateServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get locate service client: %w", err)
	}

	// 用户 ID 作为 biz id 放入 context。
	// 作为一致性哈希的 key。
	ctx = client.ContextWithBizId(ctx, uid)
	resp, err := grpcClient.LocateUser(ctx, &nodev1.LocateUserRequest{
		UserId: uid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to locate user: %w", err)
	}

	// 用户不在线直接返回空列表。
	if len(resp.Locations) == 0 {
		return nil, nil
	}

	addrs := make([]string, 0, len(resp.Locations))
	for _, location := range resp.Locations {
		addrs = append(addrs, location.ServiceAddr)
	}
	return addrs, nil
}

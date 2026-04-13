package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xpool"
	commonv1 "github.com/jrmarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	nodev1 "github.com/jrmarcco/synp-api/api/go/node/v1"
	"github.com/jrmarcco/xgrpc/client"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// TODO: 后续建议按照业务分级配置超时时间
const defaultPushTimeout = 800 * time.Millisecond

type MessageService interface {
	ProcessUpstreamMsg(ctx context.Context, upstreamMsg *xmq.Message) error
}

var _ MessageService = (*DefaultMessageService)(nil)

type DefaultMessageService struct {
	rdb         redis.Cmdable
	taskPool    xpool.TaskPool
	messageRepo repo.MessageRepo

	pushServiceClients   *client.Manager[messagev1.PushServiceClient]
	locateServiceClients *client.Manager[nodev1.LocateServiceClient]
	logger               *zap.Logger
}

func NewDefaultMessageService(
	rdb redis.Cmdable,
	taskPool xpool.TaskPool,
	messageRepo repo.MessageRepo,
	pushServiceClients *client.Manager[messagev1.PushServiceClient],
	locateServiceClients *client.Manager[nodev1.LocateServiceClient],
	logger *zap.Logger,
) *DefaultMessageService {
	return &DefaultMessageService{
		rdb:         rdb,
		taskPool:    taskPool,
		messageRepo: messageRepo,

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
	// 1.查询目标用户所在节点。
	addrs, err := s.locateUser(ctx, msg.DestinationID)
	if err != nil {
		return fmt.Errorf("failed to push single message to user: %w", err)
	}

	for _, addr := range addrs {
		// 2.使用线程池进行消息推送。
		s.logger.Debug("pushing single message to user", zap.Uint64("user_id", msg.DestinationID), zap.String("addr", addr))

		// 推荐在这里使用 WithTaskExecTimeoutInContext 设置超时时间。
		submitCtx := xpool.WithTaskExecTimeoutInContext(ctx, defaultPushTimeout)
		err := s.taskPool.Submit(submitCtx, xpool.TaskFunc(func(ctx context.Context) error {
			return s.pushMsg(ctx, addr, msg)
		}))
		if err != nil {
			return fmt.Errorf("failed to submit push message task: %w", err)
		}
	}
	return nil
}

// pushGroupMsg 推送群聊消息。
//
//nolint:unused // TODO: implement
func (s *DefaultMessageService) pushGroupMsg(_ context.Context, _ *domain.Message) error {
	return nil
}

// locateUser 定位用户所在的网关地址列表。
// 这里只需要地址，推送消息通过地址定向调用。
func (s *DefaultMessageService) locateUser(ctx context.Context, uid uint64) ([]string, error) {
	// TODO: 这里需要优化，不要硬编码服务名称。
	grpcClient, err := s.locateServiceClients.Get("synp-gateway")
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

func (s *DefaultMessageService) pushMsg(ctx context.Context, addr string, msg *domain.Message) error {
	// TODO: 这里需要优化，不要硬编码服务名称。
	grpcClient, err := s.pushServiceClients.GetByAddr("synp-gateway", addr)
	if err != nil {
		return fmt.Errorf("failed to get push service client: %w", err)
	}

	pushMsg := &messagev1.PushMessage{
		MessageId:     msg.ID,
		BusinessId:    msg.DestinationID,
		ReceiverId:    msg.DestinationID,
		SerializeType: commonv1.SerializeType_SERIALIZE_TYPE_PROTOBUF,
		Body:          msg.Content,
	}

	resp, err := grpcClient.Push(ctx, &messagev1.PushRequest{
		Message:     pushMsg,
		ShouldRetry: true,
	})
	if err != nil {
		return fmt.Errorf("failed to call the push service grpc api: %w", err)
	}

	if resp.Status != messagev1.PushStatus_PUSH_STATUS_SUCCESS {
		return fmt.Errorf("failed to push message: %s", resp.ErrorMessage)
	}

	return nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
	nodev1 "github.com/jrmarcco/synp-api/api/go/node/v1"
	"github.com/jrmarcco/xgrpc/client"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	AddUser(ctx context.Context, user domain.BizUser) error
	LocateUser(ctx context.Context, userID uint64) ([]string, error)
}

var _ UserService = (*DefaultUserService)(nil)

type DefaultUserService struct {
	bizUserRepo     repo.UserRepo
	userContactRepo repo.ContactRepo

	locateServiceClients *client.Manager[nodev1.LocateServiceClient]
}

func NewDefaultUserService(
	bizUserRepo repo.UserRepo,
	userContactRepo repo.ContactRepo,
	locateServiceClients *client.Manager[nodev1.LocateServiceClient],
) *DefaultUserService {
	return &DefaultUserService{
		bizUserRepo:     bizUserRepo,
		userContactRepo: userContactRepo,

		locateServiceClients: locateServiceClients,
	}
}

func (s *DefaultUserService) AddUser(ctx context.Context, user domain.BizUser) error {
	// 生成密码。
	passwd, err := bcrypt.GenerateFromPassword([]byte("hermet@2026"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	user.Passwd = string(passwd)

	// 设置用户状态。
	user.UserStatus = domain.UserStatusActive

	// 设置版本控制。
	user.InfoVer = 1

	// 设置时间戳。
	user.CreatedAt = time.Now().UnixMilli()
	user.UpdatedAt = user.CreatedAt

	// 添加用户。
	if err := s.bizUserRepo.Create(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *DefaultUserService) LocateUser(ctx context.Context, userID uint64) ([]string, error) {
	grpcClient, err := s.locateServiceClients.Get("synp-gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to get locate service client: %w", err)
	}

	// 用户 ID 作为 biz id 放入 context。
	// 作为一致性哈希的 key。
	ctx = client.ContextWithBizId(ctx, userID)
	resp, err := grpcClient.LocateUser(ctx, &nodev1.LocateUserRequest{
		UserId: userID,
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

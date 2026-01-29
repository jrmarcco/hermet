package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	AddUser(ctx context.Context, user domain.BizUser) error
}

var _ UserService = (*DefaultUserService)(nil)

type DefaultUserService struct {
	bizUserRepo     repo.UserRepo
	userContactRepo repo.ContactRepo
}

func NewDefaultUserService(
	bizUserRepo repo.UserRepo,
	userContactRepo repo.ContactRepo,
) *DefaultUserService {
	return &DefaultUserService{
		bizUserRepo:     bizUserRepo,
		userContactRepo: userContactRepo,
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
	_, err = s.bizUserRepo.Create(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/repo"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	SignIn(ctx context.Context, account, accountType, credential string) (user xgin.ContextUser, err error)
	GenerateToken(ctx context.Context, user xgin.ContextUser) (accessToken, refreshToken string, err error)
	VerifyRefreshToken(ctx context.Context, refreshToken string) (user xgin.ContextUser, err error)
}

const (
	accountTypeEmail = "email"

	// DefaultBusinessID 默认业务 ID。
	// 当前为单租户部署，business_id 以固定常量透传 ( 契约要求网关盖章该字段 )，
	// 多租户化时需要把 business 维度引入用户与会话模型。
	DefaultBusinessID = 1
)

var _ AuthService = (*DefaultAuthService)(nil)

type DefaultAuthService struct {
	userRepo repo.UserRepo

	// signer 签发网关连接鉴权 token ( 同时用于后端 HTTP API 鉴权 )。
	signer *auth.RS256Signer
	// rtManager 签发与校验 refresh token ( 仅后端自身使用 )。
	rtManager xjwt.Manager[authv1.JwtPayload]
}

func NewDefaultAuthService(
	userRepo repo.UserRepo,

	signer *auth.RS256Signer,
	rtManager xjwt.Manager[authv1.JwtPayload],
) *DefaultAuthService {
	return &DefaultAuthService{
		userRepo: userRepo,

		signer:    signer,
		rtManager: rtManager,
	}
}

func (s *DefaultAuthService) SignIn(ctx context.Context, account, accountType, credential string) (user xgin.ContextUser, err error) {
	switch accountType {
	case accountTypeEmail:
		return s.signInByEmail(ctx, account, credential)
	default:
		return xgin.ContextUser{}, errs.ErrInvalidAccountType
	}
}

func (s *DefaultAuthService) signInByEmail(ctx context.Context, account, credential string) (xgin.ContextUser, error) {
	user, err := s.userRepo.FindByEmail(ctx, account)
	if err != nil {
		return xgin.ContextUser{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(credential)); err != nil {
		return xgin.ContextUser{}, errs.ErrInvalidUser
	}

	return xgin.ContextUser{
		BID: DefaultBusinessID,
		SID: uuid.NewString(),
		UID: user.ID,
	}, nil
}

func (s *DefaultAuthService) GenerateToken(_ context.Context, user xgin.ContextUser) (accessToken, refreshToken string, err error) {
	if user.BID == 0 {
		user.BID = DefaultBusinessID
	}

	at, err := s.signer.Sign(user.BID, user.UID, user.SID)
	if err != nil {
		return "", "", err
	}

	rt, err := s.rtManager.Encrypt(authv1.JwtPayload{
		BusinessId: user.BID,
		UserId:     user.UID,
		SessionId:  user.SID,
	})
	if err != nil {
		return "", "", err
	}
	return at, rt, nil
}

func (s *DefaultAuthService) VerifyRefreshToken(_ context.Context, refreshToken string) (user xgin.ContextUser, err error) {
	decrypted, err := s.rtManager.Decrypt(refreshToken)
	if err != nil {
		return xgin.ContextUser{}, err
	}
	return xgin.ContextUser{
		BID: decrypted.Data.BusinessId,
		UID: decrypted.Data.UserId,
		SID: decrypted.Data.SessionId,
	}, nil
}

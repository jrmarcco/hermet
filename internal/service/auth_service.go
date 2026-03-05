package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jrmarcco/hermet/internal/errs"
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

const accountTypeEmail = "email"

var _ AuthService = (*DefaultAuthService)(nil)

type DefaultAuthService struct {
	userRepo repo.UserRepo

	atManager xjwt.Manager[authv1.JwtPayload]
	rtManager xjwt.Manager[authv1.JwtPayload]
}

func NewDefaultAuthService(
	userRepo repo.UserRepo,

	atManager xjwt.Manager[authv1.JwtPayload],
	rtManager xjwt.Manager[authv1.JwtPayload],
) *DefaultAuthService {
	return &DefaultAuthService{
		userRepo: userRepo,

		atManager: atManager,
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
		SID: uuid.NewString(),
		UID: user.ID,
	}, nil
}

func (s *DefaultAuthService) GenerateToken(_ context.Context, user xgin.ContextUser) (accessToken, refreshToken string, err error) {
	at, err := s.atManager.Encrypt(authv1.JwtPayload{
		BizId:     user.BID,
		UserId:    user.UID,
		SessionId: user.SID,
	})
	if err != nil {
		return "", "", err
	}
	rt, err := s.rtManager.Encrypt(authv1.JwtPayload{
		BizId:     user.BID,
		UserId:    user.UID,
		SessionId: user.SID,
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
		BID: decrypted.Data.BizId,
		UID: decrypted.Data.UserId,
		SID: decrypted.Data.SessionId,
	}, nil
}

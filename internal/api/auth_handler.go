package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/jrmarcco/hermet/internal/service"
)

var _ xgin.RouteRegistry = (*AuthHandler)(nil)

// AuthHandler 认证 HTTP Handler。
type AuthHandler struct {
	xsession.Handler

	svc service.AuthService
}

func NewAuthHandler(handler xsession.Handler, svc service.AuthService) *AuthHandler {
	return &AuthHandler{
		Handler: handler,
		svc:     svc,
	}
}

func (h *AuthHandler) Register(engine *gin.Engine) {
	authV1 := engine.Group("api/v1/auth")

	authV1.Handle(http.MethodPost, "/sign-in", xgin.B(h.SignIn))
	authV1.Handle(http.MethodPost, "/refresh-token", xgin.B(h.RefreshToken))
	authV1.Handle(http.MethodPost, "/sign-out", xgin.W(h.SignOut))
}

type signInReq struct {
	Account     string `json:"account"`
	AccountType string `json:"accountType"`
	Credential  string `json:"credential"`
}

type tokenResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (h *AuthHandler) SignIn(ctx *gin.Context, req signInReq) (xgin.R, error) {
	au, err := h.svc.SignIn(ctx, req.Account, req.AccountType, req.Credential)
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return xgin.R{}, fmt.Errorf("%w: user not found", errs.ErrUnauthorized)
		}
		return xgin.R{}, err
	}

	// 创建 seesion。
	err = h.CreateSession(ctx, au.SID, au.UID)
	if err != nil {
		return xgin.R{}, err
	}

	at, st, err := h.svc.GenerateToken(ctx, au)
	if err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: tokenResp{
			AccessToken:  at,
			RefreshToken: st,
		},
	}, nil
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *AuthHandler) RefreshToken(ctx *gin.Context, req refreshTokenRequest) (xgin.R, error) {
	au, err := h.svc.VerifyRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return xgin.R{}, err
	}

	// 校验 session。
	err = h.CheckSession(ctx, au.SID, au.UID)
	if err != nil {
		return xgin.R{}, fmt.Errorf("%w: %w", errs.ErrUnauthorized, err)
	}

	// 刷新 session 过期时间。
	err = h.RefreshSession(ctx, au.SID)
	if err != nil {
		// 刷新失败不中断流程，记录日志即可。
		logging.For("user-handler").Error("failed to refresh session", "error", err)
	}

	// 重新生成 access token 和 refresh token。
	at, st, err := h.svc.GenerateToken(ctx, au)
	if err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: tokenResp{
			AccessToken:  at,
			RefreshToken: st,
		},
	}, nil
}

func (h *AuthHandler) SignOut(ctx *gin.Context) (xgin.R, error) {
	val, ok := ctx.Get(xgin.ContextKeyAuthUser)
	if !ok {
		return xgin.R{}, fmt.Errorf("%w: user is not signed in", errs.ErrUnauthorized)
	}

	au, ok := val.(xgin.ContextUser)
	if !ok {
		return xgin.R{}, fmt.Errorf("%w: user is not signed in", errs.ErrUnauthorized)
	}

	go func() {
		// 异步清理 session，避免阻塞主线程。
		// 这里必须使用 context.Background()，ctx 会在请求结束后取消。
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := h.ClearSession(ctx, au.SID); err != nil {
			logging.For("user-handler").Error("failed to clear session", "error", err)
		}
	}()
	return xgin.R{
		Code: http.StatusOK,
		Msg:  "signed out",
	}, nil
}

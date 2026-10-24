package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
)

var _ xgin.RouteRegistry = (*JwksHandler)(nil)

// JwksHandler 发布网关连接鉴权的公钥集 ( RFC 7517 )。
// 网关定期拉取该端点缓存公钥，token header 的 kid 用于密钥轮换。
type JwksHandler struct {
	jwks *auth.JWKS
}

func NewJwksHandler(jwks *auth.JWKS) *JwksHandler {
	return &JwksHandler{jwks: jwks}
}

func (h *JwksHandler) Register(engine *gin.Engine) {
	engine.GET("/.well-known/jwks.json", h.handleJwks)
}

func (h *JwksHandler) handleJwks(ctx *gin.Context) {
	body, err := h.jwks.MarshalJSON()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ctx.Data(http.StatusOK, "application/json", body)
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/jrmarcco/jit/xset"
)

var _ xgin.HandlerFuncBuilder = (*JwtBuilder)(nil)

type JwtBuilder struct {
	handler  xsession.Handler
	verifier *auth.RS256Verifier
	ignores  xset.Set[string]
}

func NewJwtBuilder(
	handler xsession.Handler,
	verifier *auth.RS256Verifier,
	ignores xset.Set[string],
) *JwtBuilder {
	return &JwtBuilder{
		handler:  handler,
		verifier: verifier,
		ignores:  ignores,
	}
}

func (b *JwtBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if b.ignores != nil && b.ignores.Exist(ctx.Request.URL.Path) {
			ctx.Next()
			return
		}

		// CORS 预检 ( OPTIONS ) 不携带凭证，交由 CORS 中间件应答，
		// 不能在鉴权层被 401 拦截 ( 否则浏览器直连后端时预检失败 )。
		if ctx.Request.Method == http.MethodOptions {
			ctx.Next()
			return
		}

		token := strings.TrimPrefix(ctx.GetHeader(xgin.HeaderNameAccessToken), "Bearer ")
		if token == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := b.verifier.Verify(token)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		ctx.Set(xgin.ContextKeyAuthUser, xgin.ContextUser{
			BID: claims.BusinessID,
			UID: claims.UserID,
			SID: claims.SessionID,
		})
		ctx.Next()
	}
}

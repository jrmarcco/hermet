package xgin

import "github.com/gin-gonic/gin"

//go:generate mockgen -source=types.go -destination=mock/xgin.mock.go -package=xginmock -typed RouteRegistry,HandlerFuncBuilder

const (
	HeaderNameAccessToken = "x-access-token"
	ContextKeyAuthUser    = "auth-user"
)

// RouteRegistry 是注册路由的接口。
// HTTP Handler 通过实现这个接口，可以注册路由到 gin 引擎中。
type RouteRegistry interface {
	Register(engine *gin.Engine)
}

// R 是 HTTP Handler统一返回结构体。
type R struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	Data any    `json:"data"`
}

type ContextUser struct {
	BID uint64 `json:"bid"` // biz id
	UID uint64 `json:"uid"` // user id
	SID string `json:"sid"` // session id
}

// HandlerFuncBuilder 是构建 gin.HandlerFunc 的接口。
type HandlerFuncBuilder interface {
	Build() gin.HandlerFunc
}

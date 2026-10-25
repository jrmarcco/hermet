package xgin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/errs"
)

// W 封装最基础的 gin.handlerFunc。
func W(bizFunc func(*gin.Context) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		r, err := bizFunc(ctx)
		writeRes(ctx, r, err)
	}
}

// U 封装包含用户登录信息的 gin.HandlerFunc。
func U(bizFunc func(*gin.Context, ContextUser) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		au, ok := extractAuthUser(ctx)
		if !ok {
			return
		}
		r, err := bizFunc(ctx, au)
		writeRes(ctx, r, err)
	}
}

// B 封装从请求体获取参数的 gin.HandlerFunc。
func B[Req any](bizFunc func(*gin.Context, Req) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindJSON(&req); err != nil {
			slog.Error("failed to bind request", slog.Any("err", err))
			return
		}

		r, err := bizFunc(ctx, req)
		writeRes(ctx, r, err)
	}
}

// Q 封装从 url query 上获取参数的 gin.HandlerFunc。
func Q[Req any](bizFunc func(*gin.Context, Req) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindQuery(&req); err != nil {
			slog.Error("failed to bind request from query", slog.Any("err", err))
			return
		}

		r, err := bizFunc(ctx, req)
		writeRes(ctx, r, err)
	}
}

// P 封装从 url path 上获取参数的 gin.HandlerFunc。
func P[Req any](bizFunc func(*gin.Context, Req) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindUri(&req); err != nil {
			slog.Error("failed to bind request from uri", slog.Any("err", err))
			return
		}

		r, err := bizFunc(ctx, req)
		writeRes(ctx, r, err)
	}
}

// WU 封装包含用户登录信息的 gin.handlerFunc。
func WU(bizFunc func(*gin.Context, ContextUser) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		au, ok := extractAuthUser(ctx)
		if !ok {
			return
		}

		r, err := bizFunc(ctx, au)
		writeRes(ctx, r, err)
	}
}

// BU 封装从请求体获取参数的且包含用户登录信息 gin.HandlerFunc。
func BU[Req any](bizFunc func(*gin.Context, Req, ContextUser) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindJSON(&req); err != nil {
			slog.Error("failed to bind request", slog.Any("err", err))
			return
		}
		au, ok := extractAuthUser(ctx)
		if !ok {
			return
		}

		r, err := bizFunc(ctx, req, au)
		writeRes(ctx, r, err)
	}
}

// QU 封装从 url query 上获取参数且包含用户登录信息 gin.HandlerFunc。
func QU[Req any](bizFunc func(*gin.Context, Req, ContextUser) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindQuery(&req); err != nil {
			slog.Error("failed to bind request from query", slog.Any("err", err))
			return
		}
		au, ok := extractAuthUser(ctx)
		if !ok {
			return
		}

		r, err := bizFunc(ctx, req, au)
		writeRes(ctx, r, err)
	}
}

// PU 封装从 url path 上获取参数且包含用户登录信息 gin.HandlerFunc。
func PU[Req any](bizFunc func(*gin.Context, Req, ContextUser) (R, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.BindUri(&req); err != nil {
			slog.Error("failed to bind request from uri", slog.Any("err", err))
			return
		}
		au, ok := extractAuthUser(ctx)
		if !ok {
			return
		}

		r, err := bizFunc(ctx, req, au)
		writeRes(ctx, r, err)
	}
}

func extractAuthUser(ctx *gin.Context) (ContextUser, bool) {
	rawVal, ok := ctx.Get(ContextKeyAuthUser)
	if !ok {
		slog.Error("failed to get auth user")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return ContextUser{}, false
	}

	// 注意 gin.Context 内的值不能是 *AuthUser
	au, ok := rawVal.(ContextUser)
	if !ok {
		slog.Error("failed to get auth user")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return ContextUser{}, false
	}
	return au, true
}

func writeRes(ctx *gin.Context, res R, err error) {
	if errors.Is(err, errs.ErrUnauthorized) {
		slog.Debug("unauthorized", slog.Any("err", err))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if err != nil {
		// 客户端参数错误应答 400 并回显校验信息；
		// 其余按服务端错误应答 500，只回通用文案——
		// 原始错误 ( 可能含 SQL/内部实现细节 ) 仅进日志，不透出给客户端。
		// 错误信封与正常信封一致使用 message 字段。
		status := http.StatusInternalServerError
		message := "internal error"
		if errors.Is(err, errs.ErrInvalidParam) {
			status = http.StatusBadRequest
			message = err.Error()
		}

		slog.Error("failed to handle request", slog.Any("err", err))
		ctx.PureJSON(status, gin.H{
			"code":    status,
			"message": message,
			"data":    nil,
		})
		return
	}
	ctx.PureJSON(res.Code, res)
}

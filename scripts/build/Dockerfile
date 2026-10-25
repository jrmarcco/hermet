# syntax=docker/dockerfile:1

# ==================== 构建阶段 ====================
ARG GO_VERSION=1.27

FROM golang:${GO_VERSION}-alpine AS builder

# 代理可选注入 ( 国内网络: --build-arg GOPROXY=https://goproxy.cn,direct )，
# 不再写死进镜像层。
ARG GOPROXY=""

# 版本信息 ( make image 传入，经 ldflags 注入二进制，见 internal/pkg/buildinfo )。
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN if [ -n "$GOPROXY" ]; then go env -w GOPROXY="$GOPROXY"; fi \
    && go mod download

COPY . .
# hermet 主程序；reindex 是独立的索引重建工具，按需挂配置单独运行。
RUN go build -trimpath \
        -ldflags "-s -w \
            -X 'github.com/jrmarcco/hermet/internal/pkg/buildinfo.Version=${VERSION}' \
            -X 'github.com/jrmarcco/hermet/internal/pkg/buildinfo.Commit=${COMMIT}' \
            -X 'github.com/jrmarcco/hermet/internal/pkg/buildinfo.Date=${DATE}'" \
        -o /out/hermet ./cmd \
    && go build -trimpath -ldflags="-s -w" -o /out/reindex ./cmd/reindex

# ==================== 运行阶段 ====================
# alpine + 非 root: 体积小、带 busybox ( HEALTHCHECK 的 wget、进容器排查 )，
# ca-certificates 供 etcd/JWKS/ES 等公网 HTTPS 使用。
FROM alpine:3.21

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 hermet

WORKDIR /app

COPY --from=builder /out/hermet ./hermet
COPY --from=builder /out/reindex ./reindex

USER hermet

# 10081: 业务 HTTP; 10091: gRPC 数据面 ( SyncService，网关调用 );
# 10090: 管理面 ( /metrics /healthz /readyz /version /debug/pprof )。
EXPOSE 10081 10091 10090

STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:10090/healthz || exit 1

# 配置文件不打包进镜像，运行时必须挂载:
#   docker run -v $PWD/etc/config.yaml:/app/etc/config.yaml:ro hermet:<version>
ENTRYPOINT ["./hermet"]
CMD ["--config", "etc/config.yaml"]

# 升级 jit/xjwt 以替换自研网关 token 签发器

## 背景

网关连接鉴权 JWT 的契约 ( `synp-api proto/auth/v1/auth.proto` ) 有两个硬性要求:

1. **RS256 签名** + token header 携带 `kid` ( 公钥经 JWKS 端点发布，用于密钥轮换 )。
2. **`aud = "synp-gateway"`**，网关侧 `jwt.WithAudience` 强制校验。
3. 业务 claims ( `business_id` / `user_id` / `session_id` ) 以**顶层 snake_case 键**写入，
   网关侧解析结构见 `gateway internal/pkg/auth/jwt_validator.go` 的 `jwtClaims`。

当前 `jit/xjwt` ( v0.0.8 ) 不满足第 2、3 点:

- `ClaimsConfig` 只有 `Issuer` / `Expiration` / `JtiGenerator`，**没有 Audience**。
- `CustomClaims[T]` 把业务 payload 序列化在 `Data` 字段下 ( JSON key 为 `Data` )，
   而网关要求业务 claims 在顶层。

因此 backend 在 `internal/pkg/auth/gateway_token.go` 自研了签发器/验证器
( `RS256Signer` / `RS256Verifier` )，两者合计约 200 行，仅使用 `golang-jwt/jwt/v5`。

## 升级 jit/xjwt 的改造点

在 jit 仓库完成以下改动后，即可用 xjwt 替换自研实现:

### 1. ClaimsConfig 增加 Audience

```go
// xjwt/types.go
type ClaimsConfig struct {
    Issuer       string
    Audience     []string      // 新增: RFC 7519 aud claim
    Expiration   time.Duration
    JtiGenerator func() string
}

func WithAudience(audience ...string) option.Opt[ClaimsConfig] {
    return func(cfg *ClaimsConfig) {
        cfg.Audience = audience
    }
}
```

各 Manager 的 `Encrypt` 在构造 `jwt.RegisteredClaims` 时补上:

```go
RegisteredClaims: jwt.RegisteredClaims{
    Issuer:    m.config.Issuer,
    Audience:  jwt.ClaimStrings(m.config.Audience), // 新增
    IssuedAt:  jwt.NewNumericDate(now),
    ExpiresAt: jwt.NewNumericDate(now.Add(m.config.Expiration)),
    ID:        m.config.JtiGenerator(),
},
```

### 2. ( 可选 ) 支持自定义 claims 布局或 kid header

- 若要满足"业务 claims 在顶层"的契约，`CustomClaims[T]` 需要支持把 `T` 平铺到顶层
  ( 例如提供一个 `FlatClaims[T]` 类型，或允许调用方直接传入自定义 `jwt.Claims` 实现 )。
- `Encrypt` 目前不支持设置 token header ( `kid` )。可以增加
  `WithHeader(map[string]any)` 之类的选项，或在 `RSAManager` 上暴露 kid 配置。

只有第 1、2 点都完成后，才能完整替换自研签发器；若只完成第 1 点，
仍需自研 claims 布局，收益有限。

## 替换步骤 ( 假设 jit 已发布支持上述能力的版本 )

1. `go.mod` 升级 `github.com/jrmarcco/jit`。
2. `internal/pkg/provider/jwt_manager.go`:
   - `newGatewayTokenSigner` 改为 `xjwt.NewRSAManagerBuilder[...](private, public).ClaimsConfig(...)`，
     带上 `WithAudience("synp-gateway")` 与顶层 claims 布局。
3. `internal/pkg/xgin/middleware/jwt.go`: `RS256Verifier` 换回
   `xjwt.Manager[...]` 的 `Decrypt`。
4. `internal/service/auth_service.go`: `signer.Sign(...)` 换回 `atManager.Encrypt(...)`。
5. 删除 `internal/pkg/auth/gateway_token.go` ( JWKS 文档构建 `jwks.go` 保留，
   它与签发器实现无关 )。

## 验证

- `GET /.well-known/jwks.json` 输出不变。
- 签发的 token 在网关侧能通过 AUTH ( RS256 + kid + aud + 顶层 claims )。
- 后端 HTTP API 鉴权 ( `x-access-token` ) 正常。

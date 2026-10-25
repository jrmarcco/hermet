// Package auth 提供网关连接鉴权 JWT ( 契约见 synp-api proto/auth/v1 ) 的签发与验证。
//
// 契约要点:
//  1. 签名算法固定 RS256，后端持有私钥签发，网关仅持公钥本地验签。
//  2. 公钥经 JWKS 端点 ( /.well-known/jwks.json ) 发布，token header 的 kid 用于密钥轮换。
//  3. 校验项: 签名、exp、aud ( 固定为网关服务标识 synp-gateway )。
//  4. 业务 claims ( business_id / user_id / session_id ) 以顶层 snake_case 键写入，
//     与网关侧的解析结构 ( gateway internal/pkg/auth/jwtClaims ) 保持一致。
//  5. business_id / user_id 在 token payload JSON 中以十进制字符串序列化
//     ( JS Number 无法无损承载 > 2^53-1 的 int64 )；解析侧同时兼容
//     字符串与数字字面量 ( 升级前签发的存量 token )。
//
// 注意: 这里没有复用 jit/xjwt，因为它不支持 aud claim 且把 payload 包在
// Data 字段下，与网关的顶层 claims 契约不兼容。待 jit 支持后可替换
// ( 见 docs/jit-xjwt-upgrade.md )。
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GatewayClaims 是网关连接鉴权 JWT 的 claims 契约。
// 注册 claims ( iss/aud/iat/exp/jti ) 由 jwt.RegisteredClaims 按标准名解析。
type GatewayClaims struct {
	jwt.RegisteredClaims

	BusinessID uint64 `json:"business_id"`
	UserID     uint64 `json:"user_id"`
	SessionID  string `json:"session_id"`
}

// MarshalJSON 将 business_id / user_id 序列化为十进制字符串。
func (c GatewayClaims) MarshalJSON() ([]byte, error) {
	type alias GatewayClaims

	return json.Marshal(struct {
		alias

		BusinessID string `json:"business_id"`
		UserID     string `json:"user_id"`
	}{
		alias: alias(c),

		BusinessID: strconv.FormatUint(c.BusinessID, 10),
		UserID:     strconv.FormatUint(c.UserID, 10),
	})
}

// UnmarshalJSON 兼容字符串与数字字面量两种 ID 形态。
func (c *GatewayClaims) UnmarshalJSON(data []byte) error {
	type alias GatewayClaims

	aux := struct {
		*alias

		BusinessID *flexibleU64 `json:"business_id"`
		UserID     *flexibleU64 `json:"user_id"`
	}{alias: (*alias)(c)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.BusinessID != nil {
		c.BusinessID = uint64(*aux.BusinessID)
	}
	if aux.UserID != nil {
		c.UserID = uint64(*aux.UserID)
	}
	return nil
}

// flexibleU64 兼容十进制字符串与数字字面量的 uint64。
type flexibleU64 uint64

func (u *flexibleU64) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*u = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*u = flexibleU64(val)
	return nil
}

var (
	_ TokenSigner   = (*RS256Signer)(nil)
	_ TokenVerifier = (*RS256Verifier)(nil)
)

// TokenSigner 签发网关连接鉴权 JWT。
type TokenSigner interface {
	// Sign 签发 token。token 同时用于后端 HTTP API 鉴权与网关 WS 首帧 AUTH。
	Sign(businessID, userID uint64, sessionID string) (string, error)
}

// TokenVerifier 验证网关连接鉴权 JWT ( 后端 HTTP API 鉴权用 )。
type TokenVerifier interface {
	// Verify 验签并返回 claims。校验项: 签名、exp。
	Verify(token string) (*GatewayClaims, error)
}

// RS256Signer 按 auth/v1 契约签发 RS256 token。
type RS256Signer struct {
	verifier *RS256Verifier

	privateKey interface{} // *rsa.PrivateKey

	issuer     string
	audience   string
	expiration time.Duration

	newJti func() string
}

// SignerConfig 是签发器配置。
type SignerConfig struct {
	// PrivatePEM / PublicPEM 为 PKCS#8 ( 私钥 ) 与 PKIX ( 公钥 ) PEM。
	PrivatePEM string
	PublicPEM  string

	// Kid 出现在 token header 与 JWKS 中，用于密钥轮换。
	Kid string

	Issuer     string
	Audience   string
	Expiration time.Duration
}

// NewRS256Signer 创建签发器，同时提供验签与 JWKS 能力。
func NewRS256Signer(cfg SignerConfig, newJti func() string) (*RS256Signer, error) {
	verifier, err := NewRS256Verifier(cfg.PublicPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create rsa verifier: %w", err)
	}
	verifier.WithKid(cfg.Kid)

	if cfg.PrivatePEM == "" {
		return nil, errors.New("private key pem is required")
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(cfg.PrivatePEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rsa private key: %w", err)
	}

	if cfg.Kid == "" {
		return nil, errors.New("kid is required")
	}
	if cfg.Issuer == "" || cfg.Audience == "" {
		return nil, errors.New("issuer and audience are required")
	}
	if cfg.Expiration <= 0 {
		return nil, errors.New("expiration must be positive")
	}
	if newJti == nil {
		return nil, errors.New("jti generator is required")
	}

	return &RS256Signer{
		verifier: verifier,

		privateKey: privateKey,

		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		expiration: cfg.Expiration,

		newJti: newJti,
	}, nil
}

// Sign 签发 token。
func (s *RS256Signer) Sign(businessID, userID uint64, sessionID string) (string, error) {
	now := time.Now()
	claims := &GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			ID:        s.newJti(),
		},

		BusinessID: businessID,
		UserID:     userID,
		SessionID:  sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid()

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign gateway token: %w", err)
	}
	return signed, nil
}

func (s *RS256Signer) kid() string {
	return s.verifier.Kid()
}

// Verifier 返回配套的验签器。
func (s *RS256Signer) Verifier() *RS256Verifier {
	return s.verifier
}

// RS256Verifier 使用 RSA 公钥验证网关 token。
type RS256Verifier struct {
	publicKey any    // *rsa.PublicKey
	kid       string // 所属公钥的 kid
}

// NewRS256Verifier 创建验签器。
func NewRS256Verifier(publicPEM string) (*RS256Verifier, error) {
	if publicPEM == "" {
		return nil, errors.New("public key pem is required")
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rsa public key: %w", err)
	}
	return &RS256Verifier{publicKey: publicKey}, nil
}

// WithKid 关联 kid ( JWKS 发布与 header 校验用 )。
func (v *RS256Verifier) WithKid(kid string) *RS256Verifier {
	v.kid = kid
	return v
}

// Kid 返回公钥 kid。
func (v *RS256Verifier) Kid() string {
	return v.kid
}

// Verify 验签 token。
//
// 与网关的校验差异: 网关额外校验 aud，这里只校验签名与 exp——
// token 本身由本服务签发，aud 固定，无需重复校验。
func (v *RS256Verifier) Verify(token string) (*GatewayClaims, error) {
	claims := &GatewayClaims{}

	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return v.publicKey, nil
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(defaultVerifyLeeway),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify gateway token: %w", err)
	}

	return claims, nil
}

// defaultVerifyLeeway 与网关侧保持一致的时钟容差。
const defaultVerifyLeeway = 60 * time.Second

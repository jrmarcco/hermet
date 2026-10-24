package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

// JWK 是 JWKS 端点返回的单个 JSON Web Key ( RFC 7517 )。
// 字段命名与网关侧的解析结构 ( gateway internal/pkg/auth/jwks.JWK ) 保持一致。
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`

	// RSA 公钥参数 ( base64url 编码，无 padding )。
	N string `json:"n"`
	E string `json:"e"`
}

// JWKS 是 JWKS 文档 ( /.well-known/jwks.json 的响应体 )。
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// jwk 常量。
const (
	jwkKtyRSA   = "RSA"
	jwkUseSig   = "sig"
	jwkAlgRS256 = "RS256"
)

var (
	errNilPublicKey  = errors.New("rsa public key is nil")
	errEmptyJWKSKeys = errors.New("jwks keys is empty")
)

// BuildJWKS 从验签器构建 JWKS 文档。
// 密钥轮换时新旧公钥需在 JWKS 中并存一段重叠期 ( 见 proto/auth/v1/auth.proto )。
func BuildJWKS(kid string, publicPEM string) (*JWKS, error) {
	if kid == "" {
		return nil, errors.New("kid is required")
	}

	verifier, err := NewRS256Verifier(publicPEM)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := verifier.publicKey.(*rsa.PublicKey)
	if !ok || rsaKey == nil {
		return nil, errNilPublicKey
	}

	jwk, err := newJWK(kid, rsaKey)
	if err != nil {
		return nil, err
	}

	return &JWKS{Keys: []JWK{*jwk}}, nil
}

func newJWK(kid string, key *rsa.PublicKey) (*JWK, error) {
	if key == nil || key.N == nil || key.E == 0 {
		return nil, errNilPublicKey
	}

	// 模长或指数为异常小值时视为非法密钥。
	if key.N.BitLen() < 2048 { //nolint:mnd // RSA 密钥长度下限
		return nil, fmt.Errorf("rsa key length must be at least 2048 bits, got %d", key.N.BitLen())
	}

	return &JWK{
		Kty: jwkKtyRSA,
		Kid: kid,
		Use: jwkUseSig,
		Alg: jwkAlgRS256,

		N: base64url(key.N.Bytes()),
		E: base64url(big.NewInt(int64(key.E)).Bytes()),
	}, nil
}

// MarshalJSON 实现 json.Marshaler，输出 JWKS 文档。
func (d *JWKS) MarshalJSON() ([]byte, error) {
	if len(d.Keys) == 0 {
		return nil, errEmptyJWKSKeys
	}

	type alias JWKS
	return json.Marshal((*alias)(d))
}

// base64url 无 padding 的 base64url 编码 ( RFC 7515 )。
func base64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

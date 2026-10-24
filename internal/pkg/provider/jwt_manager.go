package provider

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type jwtFxResult struct {
	fx.Out

	Signer *auth.RS256Signer
	Jwks   *auth.JWKS

	RtManager xjwt.Manager[authv1.JwtPayload] `name:"refresh-token-manager"`
}

func newJwtManager() (jwtFxResult, error) {
	gwCfg, err := loadJwtConfig("jwt.gateway")
	if err != nil {
		return jwtFxResult{}, err
	}

	signer, err := auth.NewRS256Signer(
		auth.SignerConfig{
			PrivatePEM: gwCfg.Private,
			PublicPEM:  gwCfg.Public,

			Kid: gwCfg.Kid,

			Issuer:     gwCfg.Issuer,
			Audience:   gwCfg.Audience,
			Expiration: gwCfg.Expiration,
		},
		newJti,
	)
	if err != nil {
		return jwtFxResult{}, fmt.Errorf("failed to create gateway token signer: %w", err)
	}

	jwks, err := auth.BuildJWKS(gwCfg.Kid, gwCfg.Public)
	if err != nil {
		return jwtFxResult{}, fmt.Errorf("failed to build jwks: %w", err)
	}

	rtManager, err := newRefreshTokenManager()
	if err != nil {
		return jwtFxResult{}, err
	}

	return jwtFxResult{
		Signer:    signer,
		Jwks:      jwks,
		RtManager: rtManager,
	}, nil
}

// newRefreshTokenManager 创建用于 Refresh Token 的 Manager。
// Refresh token 仅由后端自身签发与校验 ( 网关不感知 )，沿用 jit/xjwt。
func newRefreshTokenManager() (xjwt.Manager[authv1.JwtPayload], error) {
	cfg, err := loadJwtConfig("jwt.refresh")
	if err != nil {
		return nil, err
	}

	claimsCfg := xjwt.NewClaimsConfig(
		xjwt.WithExpiration(cfg.Expiration),
		xjwt.WithIssuer(cfg.Issuer),
	)

	manager, err := xjwt.NewEd25519ManagerBuilder[authv1.JwtPayload](cfg.Private, cfg.Public).
		ClaimsConfig(claimsCfg).
		Build()
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// newJti 生成 token 唯一标识 ( 用于审计与潜在的吊销场景 )。
func newJti() string {
	return uuid.NewString()
}

// loadJwtConfig 加载配置。
func loadJwtConfig(key string) (*jwtConfig, error) {
	cfg := &jwtConfig{}
	if err := viper.UnmarshalKey(key, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

type jwtConfig struct {
	Kid string `mapstructure:"kid"`

	Issuer     string        `mapstructure:"issuer"`
	Audience   string        `mapstructure:"audience"`
	Expiration time.Duration `mapstructure:"expiration"`
	Private    string        `mapstructure:"private"`
	Public     string        `mapstructure:"public"`
}

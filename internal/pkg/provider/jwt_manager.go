package provider

import (
	"time"

	"github.com/jrmarcco/jit/xjwt"
	authv1 "github.com/jrmarcco/synp-api/api/go/auth/v1"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type jwtFxResult struct {
	fx.Out

	AtManager xjwt.Manager[authv1.JwtPayload] `name:"access-token-manager"`
	RtManager xjwt.Manager[authv1.JwtPayload] `name:"refresh-token-manager"`
}

func newJwtManager() (jwtFxResult, error) {
	atManager, err := newAccessTokenManager()
	if err != nil {
		return jwtFxResult{}, err
	}
	rtManager, err := newRefreshTokenManager()
	if err != nil {
		return jwtFxResult{}, err
	}

	return jwtFxResult{
		AtManager: atManager,
		RtManager: rtManager,
	}, nil
}

// newAccessTokenManager 创建用于 Access Token 的 Manager。
func newAccessTokenManager() (xjwt.Manager[authv1.JwtPayload], error) {
	cfg, err := loadJwtConfig("jwt.access")
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

// newRefreshTokenManager 创建用于 Refresh Token 的 Manager。
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

// loadJwtConfig 加载配置。
func loadJwtConfig(key string) (*jwtConfig, error) {
	cfg := &jwtConfig{}
	if err := viper.UnmarshalKey(key, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

type jwtConfig struct {
	Issuer     string        `mapstructure:"issuer"`
	Expiration time.Duration `mapstructure:"expiration"`
	Private    string        `mapstructure:"private"`
	Public     string        `mapstructure:"public"`
}

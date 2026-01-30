package provider

import (
	"time"

	"github.com/jrmarcco/hermet/internal/api/session"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func newRedisSessionHandler(rdb redis.Cmdable) (xsession.Handler, error) {
	var expiration time.Duration
	if err := viper.UnmarshalKey("hermet.session.expiration", &expiration); err != nil {
		return nil, err
	}
	return session.NewRedisSessionHandler(rdb, expiration)
}

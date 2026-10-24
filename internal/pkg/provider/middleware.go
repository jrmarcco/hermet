package provider

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/auth"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/middleware"
	"github.com/jrmarcco/hermet/internal/pkg/xgin/xsession"
	"github.com/jrmarcco/jit/xset"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func newCorsBuilder() (*middleware.CorsBuilder, error) {
	type config struct {
		MaxAge    time.Duration `mapstructure:"max_age"`
		Hostnames []string      `mapstructure:"hostnames"`
	}
	cfg := config{}
	if err := viper.UnmarshalKey("hermet.cors", &cfg); err != nil {
		return nil, err
	}

	builder := middleware.NewCorsBuilder().
		AllowCredentials(true).
		AllowMethods([]string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions}).
		AllowHeaders([]string{"Content-Length", "Content-Type", "Authorization", "Accept", "Origin", xgin.HeaderNameAccessToken}).
		MaxAge(cfg.MaxAge).
		AllowOriginFunc(func(origin string) bool {
			if origin == "" {
				return false
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			reqHostname := u.Hostname()
			return slices.Contains(cfg.Hostnames, reqHostname)
		})
	return builder, nil
}

type jwtBuilderFxParams struct {
	fx.In

	Handler xsession.Handler
	Signer  *auth.RS256Signer
}

func newJwtBuilder(params jwtBuilderFxParams) (*middleware.JwtBuilder, error) {
	var ignores []string
	if err := viper.UnmarshalKey("hermet.ignores", &ignores); err != nil {
		return nil, err
	}

	ts, err := xset.NewTreeSet(strings.Compare)
	if err != nil {
		return nil, err
	}
	for _, ignore := range ignores {
		ts.Add(ignore)
	}
	return middleware.NewJwtBuilder(params.Handler, params.Signer.Verifier(), ts), nil
}

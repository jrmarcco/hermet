package app

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

type App struct {
	svr    *http.Server
	logger *zap.Logger
}

func (app *App) Start() error {
	go func() {
		if err := app.svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Fatal("[app] failed to start http server", zap.Error(err))
		}
	}()

	app.logger.Info("[app] http server started")
	return nil
}

func (app *App) Stop(ctx context.Context) error {
	if err := app.svr.Shutdown(ctx); err != nil {
		app.logger.Error("[app] failed to shutdown http server", zap.Error(err))
		return err
	}
	return nil
}

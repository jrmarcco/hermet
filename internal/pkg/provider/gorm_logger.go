package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jrmarcco/hermet/internal/pkg/logging"
	"github.com/jrmarcco/jit/xbean/option"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ logger.Interface = (*GormLogger)(nil)

// GormLogger 是 gorm logger 的装饰器。
// 使用 slog 记录 gorm 的日志 ( 与业务日志共用同一条管道 )。
type GormLogger struct {
	logLevel                  logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

func WithLogLevel(level logger.LogLevel) option.Opt[GormLogger] {
	return func(zl *GormLogger) {
		zl.logLevel = level
	}
}

func WithSlowThreshold(threshold time.Duration) option.Opt[GormLogger] {
	return func(zl *GormLogger) {
		zl.slowThreshold = threshold
	}
}

func WithIgnoreRecordNotFoundError(ignore bool) option.Opt[GormLogger] {
	return func(zl *GormLogger) {
		zl.ignoreRecordNotFoundError = ignore
	}
}

func NewGormLogger(opts ...option.Opt[GormLogger]) *GormLogger {
	const defaultSlowThreshold = 100 * time.Millisecond
	gl := &GormLogger{
		logLevel:      logger.Warn,
		slowThreshold: defaultSlowThreshold,
	}

	option.Apply(gl, opts...)
	return gl
}

func (gl *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *gl
	newLogger.logLevel = level
	return &newLogger
}

func (gl *GormLogger) Info(_ context.Context, str string, args ...interface{}) {
	if gl.logLevel < logger.Info {
		return
	}
	logging.For("gorm").Info(fmt.Sprintf(str, args...))
}

func (gl *GormLogger) Warn(_ context.Context, str string, args ...interface{}) {
	if gl.logLevel < logger.Warn {
		return
	}
	logging.For("gorm").Warn(fmt.Sprintf(str, args...))
}

func (gl *GormLogger) Error(_ context.Context, str string, args ...interface{}) {
	if gl.logLevel < logger.Error {
		return
	}
	logging.For("gorm").Error(fmt.Sprintf(str, args...))
}

func (gl *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if gl.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && gl.logLevel >= logger.Error && (!gl.ignoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		logging.For("gorm").Error("gorm query error", "sql", sql, "elapsed", elapsed, "rows", rows, "error", err)
	case gl.slowThreshold != 0 && elapsed > gl.slowThreshold && gl.logLevel >= logger.Warn:
		logging.For("gorm").Warn("gorm slow query", "sql", sql, "elapsed", elapsed, "rows", rows)
	case gl.logLevel >= logger.Info:
		logging.For("gorm").Info("gorm query", "sql", sql, "elapsed", elapsed, "rows", rows)
	}
}

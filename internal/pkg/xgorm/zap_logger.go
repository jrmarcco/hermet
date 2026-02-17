package xgorm

import (
	"context"
	"errors"
	"time"

	"github.com/jrmarcco/jit/bean/option"
	"go.uber.org/zap" //nolint:depguard // 替换 gorm 默认日志库。
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ logger.Interface = (*ZapLogger)(nil)

// ZapLogger gorm logger 的 zap 装饰器。
type ZapLogger struct {
	zLogger                   *zap.Logger
	logLevel                  logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

func WithLogLevel(level logger.LogLevel) option.Opt[ZapLogger] {
	return func(zl *ZapLogger) {
		zl.logLevel = level
	}
}

func WithSlowThreshold(threshold time.Duration) option.Opt[ZapLogger] {
	return func(zl *ZapLogger) {
		zl.slowThreshold = threshold
	}
}

func WithIgnoreRecordNotFoundError(ignore bool) option.Opt[ZapLogger] {
	return func(zl *ZapLogger) {
		zl.ignoreRecordNotFoundError = ignore
	}
}

func NewZapLogger(zLogger *zap.Logger, opts ...option.Opt[ZapLogger]) *ZapLogger {
	const defaultSlowThreshold = 100 * time.Millisecond
	zl := &ZapLogger{
		zLogger:                   zLogger,
		logLevel:                  logger.Warn,
		slowThreshold:             defaultSlowThreshold,
		ignoreRecordNotFoundError: false,
	}

	option.Apply(zl, opts...)
	return zl
}

func (zl *ZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *zl
	newLogger.logLevel = level
	return &newLogger
}

func (zl *ZapLogger) Info(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Info {
		return
	}
	zl.zLogger.Sugar().Infof(str, args...)
}

func (zl *ZapLogger) Warn(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Warn {
		return
	}
	zl.zLogger.Sugar().Warnf(str, args...)
}

func (zl *ZapLogger) Error(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Error {
		return
	}
	zl.zLogger.Sugar().Errorf(str, args...)
}

func (zl *ZapLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if zl.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
	}

	switch {
	case err != nil && zl.logLevel >= logger.Error && (!zl.ignoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		zl.zLogger.Error("gorm query error", append(fields, zap.Error(err))...)
	case zl.slowThreshold != 0 && elapsed > zl.slowThreshold && zl.logLevel >= logger.Warn:
		zl.zLogger.Warn("gorm slow query", fields...)
	case zl.logLevel >= logger.Info:
		zl.zLogger.Info("gorm query", fields...)
	}
}

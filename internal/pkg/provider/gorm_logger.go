package provider

import (
	"context"
	"errors"
	"time"

	"github.com/jrmarcco/jit/bean/option"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ logger.Interface = (*GormLogger)(nil)

// GormLogger 是 gorm logger 的装饰器。
// 使用 zap 记录 gorm 的日志。
type GormLogger struct {
	zapLogger                 *zap.Logger
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

func NewGormLogger(zLogger *zap.Logger, opts ...option.Opt[GormLogger]) *GormLogger {
	const defaultSlowThreshold = 100 * time.Millisecond
	zl := &GormLogger{
		zapLogger:                 zLogger,
		logLevel:                  logger.Warn,
		slowThreshold:             defaultSlowThreshold,
		ignoreRecordNotFoundError: false,
	}

	option.Apply(zl, opts...)
	return zl
}

func (zl *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *zl
	newLogger.logLevel = level
	return &newLogger
}

func (zl *GormLogger) Info(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Info {
		return
	}
	zl.zapLogger.Sugar().Infof(str, args...)
}

func (zl *GormLogger) Warn(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Warn {
		return
	}
	zl.zapLogger.Sugar().Warnf(str, args...)
}

func (zl *GormLogger) Error(_ context.Context, str string, args ...interface{}) {
	if zl.logLevel < logger.Error {
		return
	}
	zl.zapLogger.Sugar().Errorf(str, args...)
}

func (zl *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
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
		zl.zapLogger.Error("gorm query error", append(fields, zap.Error(err))...)
	case zl.slowThreshold != 0 && elapsed > zl.slowThreshold && zl.logLevel >= logger.Warn:
		zl.zapLogger.Warn("gorm slow query", fields...)
	case zl.logLevel >= logger.Info:
		zl.zapLogger.Info("gorm query", fields...)
	}
}

package zlog

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
	levelPanic = "panic"
	levelFatal = "fatal"
)

type Options struct {
	CtxFunc func(context.Context)
}

type Option func(*Options)

var (
	zapLogger      *zap.Logger
	defaultOptions *Options
)

func WithCtxFunc(f func(context.Context)) Option {
	return func(options *Options) {
		options.CtxFunc = f
	}
}

// 快捷使用
func init() {
	InitLogger()
}

// InitLogger 应该在应用的main.go里首先调用，并且进程退出时调用SyncLogger()
// 默认ReleaseMode=false
// ReleaseMode=false下，就算有传入也不会打印appName,instanceName
func InitLogger(options ...Option) {
	defaultOptions = &Options{}

	for _, opt := range options {
		opt(defaultOptions)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:      "time",
		LevelKey:     "level",
		CallerKey:    "caller",
		MessageKey:   "msg",
		EncodeLevel:  zapcore.LowercaseColorLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
	}

	logger, _ := config.Build()

	zapLogger = logger.WithOptions(zap.AddCallerSkip(1))
}

func SyncLogger() error {
	return zapLogger.Sync()
}

func Debugf(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Debug(msg)
}

func Infof(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Info(msg)
}

func Warnf(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Warn(msg)
}

func Errorf(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Error(msg)
}

func Panicf(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Panic(msg)
}

func Fatalf(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Fatal(msg)
}

func Debugc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Debug(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

func Infoc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Info(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

func Warnc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Warn(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

func Errorc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Error(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

func Panicc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Panic(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

func Fatalc(ctx context.Context, format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Fatal(msg)
	if defaultOptions.CtxFunc != nil {
		defaultOptions.CtxFunc(ctx)
	}
}

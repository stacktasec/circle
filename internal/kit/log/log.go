package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debug(format any, a ...any)
	Info(format any, a ...any)
	Warn(format any, a ...any)
	Error(format any, a ...any)
	Fatal(format any, a ...any)
	Sync() error
}

var _ Logger = (*zapLogger)(nil)

type zapLogger struct {
	logger *zap.Logger
}

func (z *zapLogger) Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Debug(msg)
}

func (z *zapLogger) Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Info(msg)
}

func (z *zapLogger) Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Warn(msg)
}

func (z *zapLogger) Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Error(msg)
}

func (z *zapLogger) Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Fatal(msg)
}

func (z *zapLogger) Sync() error {
	return z.logger.Sync()
}

const (
	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
	levelFatal = "fatal"
)

type options struct {
	level      string
	stacktrace string
	json       bool

	callerSkip int
}

func (o *options) ensure() {
	switch o.level {
	case levelDebug, levelInfo, levelWarn, levelError, levelFatal:
	default:
		o.level = levelDebug
	}

	switch o.stacktrace {
	case levelDebug, levelInfo, levelWarn, levelError, levelFatal:
	default:
		o.level = levelError
	}

	if o.callerSkip == 0 {
		o.callerSkip = 1
	}
}

func convert(level string) zapcore.Level {
	switch level {
	case levelDebug:
		return zapcore.DebugLevel
	case levelInfo:
		return zapcore.InfoLevel
	case levelWarn:
		return zapcore.WarnLevel
	case levelError:
		return zapcore.ErrorLevel
	case levelFatal:
		return zapcore.FatalLevel
	default:
		panic("can not convert")
	}
}

type Option interface {
	apply(*options)
}

type logOptionFunc func(opts *options)

func (opt logOptionFunc) apply(opts *options) {
	opt(opts)
}

func WithLevel(level string) Option {
	return logOptionFunc(func(opts *options) {
		opts.level = level
	})
}

func WithStacktrace(level string) Option {
	return logOptionFunc(func(opts *options) {
		opts.stacktrace = level
	})
}

func WithJson() Option {
	return logOptionFunc(func(opts *options) {
		opts.json = true
	})
}

// internal config
func withCallerSkip(skip int) Option {
	return logOptionFunc(func(opts *options) {
		opts.callerSkip = skip
	})
}

func NewLogger(opts ...Option) Logger {
	o := &options{}

	for _, opt := range opts {
		opt.apply(o)
	}

	o.ensure()

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "time",
		CallerKey:     "caller",
		FunctionKey:   "func",
		StacktraceKey: "stacktrace",
		EncodeLevel:   zapcore.LowercaseColorLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	var encoding string
	if o.json {
		encoding = "json"
	} else {
		encoding = "console"
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(convert(o.level)),
		Encoding:         encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ := config.Build(zap.AddCallerSkip(o.callerSkip), zap.AddStacktrace(convert(o.stacktrace)))

	return &zapLogger{logger: logger}
}

var builtinLogger Logger

func init() {
	builtinLogger = NewLogger(withCallerSkip(2))
}

func Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Debug(msg)
}

func Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Info(msg)
}

func Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Warn(msg)
}

func Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Error(msg)
}

func Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Fatal(msg)
}

func SyncLogger() error {
	return builtinLogger.Sync()
}

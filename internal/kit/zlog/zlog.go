package zlog

import (
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

type options struct {
	level string
}

func (o *options) ensure() {
	switch o.level {
	case levelDebug, levelInfo, levelWarn, levelError, levelPanic, levelFatal:
	default:
		o.level = levelDebug
	}
}

func convert(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		panic("can not convert")
	}
}

type LogOption interface {
	apply(*options)
}

type logOptionFunc func(opts *options)

func (opt logOptionFunc) apply(opts *options) {
	opt(opts)
}

func WithLevel(level string) LogOption {
	return logOptionFunc(func(opts *options) {
		opts.level = level
	})
}

var (
	zapLogger  *zap.Logger
	logOptions *options
)

func init() {
	InitLogger()
}

func InitLogger(opts ...LogOption) {
	logOptions = &options{}

	for _, opt := range opts {
		opt.apply(logOptions)
	}

	logOptions.ensure()

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
		Level:            zap.NewAtomicLevelAt(convert(logOptions.level)),
	}

	logger, _ := config.Build()

	zapLogger = logger.WithOptions(zap.AddCallerSkip(1))
}

func SyncLogger() error {
	return zapLogger.Sync()
}

func Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Debug(msg)
}

func Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Info(msg)
}

func Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Warn(msg)
}

func Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Error(msg)
}

func Panic(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Panic(msg)
}

func Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	zapLogger.Fatal(msg)
}

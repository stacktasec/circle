package zap

import (
	"fmt"
	"github.com/stacktasec/circle/kit/klog/internal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
}

func NewLogger(opts ...internal.Option) *Logger {
	o := &internal.Options{}

	for _, opt := range opts {
		opt.Apply(o)
	}

	o.Ensure()

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
	if o.Json {
		encoding = "json"
	} else {
		encoding = "console"
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(convert(o.Level)),
		Encoding:         encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ := config.Build(zap.AddCallerSkip(o.CallerSkip), zap.AddStacktrace(convert(o.Stacktrace)))

	return &Logger{logger: logger}
}

func convert(level string) zapcore.Level {
	switch level {
	case internal.LevelDebug:
		return zapcore.DebugLevel
	case internal.LevelInfo:
		return zapcore.InfoLevel
	case internal.LevelWarn:
		return zapcore.WarnLevel
	case internal.LevelError:
		return zapcore.ErrorLevel
	case internal.LevelFatal:
		return zapcore.FatalLevel
	default:
		panic("can not convert")
	}
}

func (z *Logger) Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Debug(msg)
}

func (z *Logger) Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Info(msg)
}

func (z *Logger) Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Warn(msg)
}

func (z *Logger) Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Error(msg)
}

func (z *Logger) Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	z.logger.Fatal(msg)
}

func (z *Logger) Sync() error {
	return z.logger.Sync()
}

package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	InitLogger()
}

func InitLogger(opts ...Option) {
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
		StacktraceKey: "stacktrace",
		EncodeLevel:   zapcore.LowercaseColorLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(*o.level),
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ = config.Build(zap.AddCallerSkip(2))
}

func Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Debug(msg)
}

func Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Info(msg)
}

func Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Warn(msg)
}

func Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Error(msg)
}

func Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Fatal(msg)
}

func SyncLogger() error {
	return logger.Sync()
}

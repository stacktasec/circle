package log

import "go.uber.org/zap/zapcore"

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

type Option interface {
	apply(*options)
}

type optionFunc func(opts *options)

func (opt optionFunc) apply(opts *options) {
	opt(opts)
}

type options struct {
	level *zapcore.Level
}

func (o *options) ensure() {
	if o.level == nil {
		debugLevel := zapcore.DebugLevel
		o.level = &debugLevel
	}
}

func WithLevel(level string) Option {
	var zapLevel zapcore.Level
	switch level {
	case LevelDebug:
		zapLevel = zapcore.DebugLevel
	case LevelInfo:
		zapLevel = zapcore.InfoLevel
	case LevelWarn:
		zapLevel = zapcore.WarnLevel
	case LevelError:
		zapLevel = zapcore.ErrorLevel
	case LevelFatal:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.DebugLevel
	}

	return optionFunc(func(opts *options) {
		opts.level = &zapLevel
	})
}

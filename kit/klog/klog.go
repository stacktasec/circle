package klog

import (
	"github.com/stacktasec/circle/kit/klog/internal"
	"github.com/stacktasec/circle/kit/klog/zap"
)

type Logger interface {
	Debug(format any, a ...any)
	Info(format any, a ...any)
	Warn(format any, a ...any)
	Error(format any, a ...any)
	Fatal(format any, a ...any)
	Sync() error
}

var _ Logger = (*zap.Logger)(nil)

func WithLevel(level string) internal.Option {
	return internal.LogOptionFunc(func(opts *internal.Options) {
		opts.Level = level
	})
}

func WithStacktrace(level string) internal.Option {
	return internal.LogOptionFunc(func(opts *internal.Options) {
		opts.Stacktrace = level
	})
}

func WithJson() internal.Option {
	return internal.LogOptionFunc(func(opts *internal.Options) {
		opts.Json = true
	})
}

func internalWithSkip(skip int) internal.Option {
	return internal.LogOptionFunc(func(opts *internal.Options) {
		opts.CallerSkip = skip
	})
}

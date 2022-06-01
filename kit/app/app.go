package app

import (
	"github.com/stacktasec/circle/kit/app/gin"
	"github.com/stacktasec/circle/kit/app/internal"
	"net/http"
	"time"
)

type App interface {
	Provide(constructors ...any)
	Map(groups ...*internal.VersionGroup)
	Run()
}

type Request = internal.Request

var _ App = (*gin.App)(nil)

func NewGroup(mainVersion int) *internal.VersionGroup {
	if mainVersion < 0 {
		panic("main version must at least zero")
	}

	return &internal.VersionGroup{
		MainVersion: mainVersion,
	}
}

func MakeKnownError(status, message string) error {
	return internal.KnownError{
		Status:  status,
		Message: message,
	}
}

func WithAddr(addr string) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.Addr = addr
	})
}

func WithTLS(cert, key string) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.EnableTLS = true
		opts.Cert = cert
		opts.Key = key
	})
}

func WithBaseURL(url string) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.BaseURL = url
	})
}

func WithCtxTimeout(d time.Duration) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.CtxTimeout = d
	})
}

func WithSuffixes(suffixes []string) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.Suffixes = suffixes
	})
}

func WithIDInterceptor(i func(h http.Header) error) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.IDInterceptor = i
	})
}

func WithPermInterceptor(p func(h http.Header) error) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.PermInterceptor = p
	})
}

func WithRateLimit(fillInterval time.Duration, capacity, quantum int) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.EnableRateLimit = true
		opts.FillInterval = fillInterval
		opts.Capacity = int64(capacity)
		opts.Quantum = int64(quantum)
	})
}

func WithLoadLimit(maxCpu, maxMem float64) internal.AppOption {
	return internal.OptionFunc(func(opts *internal.Options) {
		opts.EnableLoadLimit = true
		opts.MaxCpuPercent = maxCpu
		opts.MaxMemPercent = maxMem
	})
}

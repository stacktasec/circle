package core

import (
	"net/http"
	"time"
)

type options struct {
	appID string

	addr string

	enableTLS  bool
	enableQUIC bool
	cert       string
	key        string

	baseURL    string
	ctxTimeout time.Duration

	idInterceptor   func(h http.Header) error
	permInterceptor func(h http.Header) error

	enableRateLimit bool
	fillInterval    time.Duration
	capacity        int64
	quantum         int64

	enableOverloadBreak bool
	maxCpuPercent       float64
	maxMemPercent       float64
}

func (o *options) ensure() {
	if o.addr == "" {
		o.addr = ":8080"
	}

	if o.ctxTimeout == 0 {
		o.ctxTimeout = time.Second * 30
	}
}

type AppOption interface {
	apply(*options)
}

type appOptionFunc func(opts *options)

func (opt appOptionFunc) apply(opts *options) {
	opt(opts)
}

func WithAppID(appID string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.appID = appID
	})
}

func WithAddr(addr string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.addr = addr
	})
}

func WithTLS(cert, key string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableTLS = true
		opts.cert = cert
		opts.key = key
	})
}

func WithQUIC(cert, key string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableQUIC = true
		opts.cert = cert
		opts.key = key
	})
}

func WithBaseURL(url string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.baseURL = url
	})
}

func WithCtxTimeout(d time.Duration) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.ctxTimeout = d
	})
}

func WithIDInterceptor(i func(h http.Header) error) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.idInterceptor = i
	})
}

func WithPermInterceptor(p func(h http.Header) error) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.permInterceptor = p
	})
}

func WithRateLimit(fillInterval time.Duration, capacity, quantum int) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableRateLimit = true
		opts.fillInterval = fillInterval
		opts.capacity = int64(capacity)
		opts.quantum = int64(quantum)
	})
}

func WithOverloadBreak(maxCpu, maxMem float64) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableOverloadBreak = true
		opts.maxCpuPercent = maxCpu
		opts.maxMemPercent = maxMem
	})
}

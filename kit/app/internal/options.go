package internal

import (
	"net/http"
	"time"
)

type AppOption interface {
	Apply(*Options)
}

type OptionFunc func(opts *Options)

func (opt OptionFunc) Apply(opts *Options) {
	opt(opts)
}

type Options struct {
	Addr string

	EnableTLS bool
	Cert      string
	Key       string

	BaseURL    string
	CtxTimeout time.Duration

	Suffixes []string

	IDInterceptor   func(h http.Header) error
	PermInterceptor func(h http.Header) error

	EnableRateLimit bool
	FillInterval    time.Duration
	Capacity        int64
	Quantum         int64

	EnableLoadLimit bool
	MaxCpuPercent   float64
	MaxMemPercent   float64
}

func (o *Options) Ensure() {
	if o.Addr == "" {
		o.Addr = ":8080"
	}

	if o.CtxTimeout == 0 {
		o.CtxTimeout = time.Second * 30
	}

	if len(o.Suffixes) == 0 {
		o.Suffixes = []string{"service", "handler", "usecase", "controller"}
	}
}

package app

import (
	"net/http"
)

type App interface {
	Load(constructors ...any)
	Map(groups ...*versionGroup)
	Run()
}

type Request interface {
	Validate() error
}

type OmittedAttribute interface {
	Omitted() bool
}

type AnonymousAttribute interface {
	Anonymous() bool
}

type versionGroup struct {
	mainVersion        int
	stableConstructors []any
	betaConstructors   []any
	alphaConstructors  []any
}

func (v *versionGroup) SetStable(services ...any) {
	v.stableConstructors = append(v.stableConstructors, services...)
}

func (v *versionGroup) SetBeta(services ...any) {
	v.betaConstructors = append(v.betaConstructors, services...)
}

func (v *versionGroup) SetAlpha(services ...any) {
	v.alphaConstructors = append(v.alphaConstructors, services...)
}

func NewGroup(mainVersion int) *versionGroup {
	if mainVersion < 0 {
		panic("main version must at least zero")
	}

	return &versionGroup{
		mainVersion: mainVersion,
	}
}

type Option interface {
	apply(*options)
}

type optionFunc func(opts *options)

func (opt optionFunc) apply(opts *options) {
	opt(opts)
}

type options struct {
	addr string

	baseURL string

	idInterceptor       func(h http.Header) error
	funcPermInterceptor func(h http.Header) error
}

func (o *options) ensure() {
	if o.addr == "" {
		o.addr = ":8080"
	}
}

func WithAddr(addr string) Option {
	return optionFunc(func(opts *options) {
		opts.addr = addr
	})
}

func WithBaseURL(url string) Option {
	return optionFunc(func(opts *options) {
		opts.baseURL = url
	})
}

func WithIDInterceptor(i func(h http.Header) error) Option {
	return optionFunc(func(opts *options) {
		opts.idInterceptor = i
	})
}

func WithFuncPermInterceptor(p func(h http.Header) error) Option {
	return optionFunc(func(opts *options) {
		opts.funcPermInterceptor = p
	})
}

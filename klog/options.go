package klog

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
	level string
}

func (o *options) ensure() {
	switch o.level {
	case LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
	default:
		o.level = LevelDebug
	}
}

func WithLevel(level string) Option {
	return optionFunc(func(opts *options) {
		opts.level = level
	})
}

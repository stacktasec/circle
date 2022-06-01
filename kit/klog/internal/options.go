package internal

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

type Option interface {
	Apply(*Options)
}

type LogOptionFunc func(opts *Options)

func (opt LogOptionFunc) Apply(opts *Options) {
	opt(opts)
}

type Options struct {
	Level      string
	Stacktrace string
	Json       bool

	CallerSkip int
}

func (o *Options) Ensure() {
	switch o.Level {
	case LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
	default:
		o.Level = LevelDebug
	}

	switch o.Stacktrace {
	case LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
	default:
		o.Stacktrace = LevelError
	}

	if o.CallerSkip == 0 {
		o.CallerSkip = 1
	}
}

package envconfig

import "strings"

const (
	DefaultFileSuffix = "_FILE"
)

type (
	options struct {
		prefix            string
		isLoadFromFile    bool
		defaultFileSuffix string
		trimSpaces        bool
	}

	Option func(o *options)
)

func defaultOptions() *options {
	return &options{
		prefix:            "",
		isLoadFromFile:    true,
		defaultFileSuffix: DefaultFileSuffix,
		trimSpaces:        true,
	}
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func (o *options) copy() *options {
	return &options{
		prefix:            o.prefix,
		isLoadFromFile:    o.isLoadFromFile,
		defaultFileSuffix: o.defaultFileSuffix,
	}
}

func WithPrefix(prefix string) Option {
	return func(o *options) {
		o.prefix = strings.ToUpper(prefix)
	}
}

// WithoutDefaultLoadingFromFiles disables loading values from files pointed by *_FILE vars.
func WithoutDefaultLoadingFromFiles() Option {
	return func(o *options) {
		o.isLoadFromFile = false
	}
}

func WithDefaultFileSuffix(suffix string) Option {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = DefaultFileSuffix
	}

	return func(o *options) {
		o.defaultFileSuffix = suffix
	}
}

// WithoutTrimSpaces disables spaces trimming around value (see strings.TrimSpace).
func WithoutTrimSpaces() Option {
	return func(o *options) {
		o.trimSpaces = false
	}
}

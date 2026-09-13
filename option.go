package sqlparser

import "github.com/viant/parsly"

type Options struct {
	structuralValidation bool
	onError              func(err error, cur *parsly.Cursor, destNode interface{}) error
}

// WithStructuralValidation rejects unbalanced parentheses and unclosed
// quotes/comments before parsing. It does not promise full dialect validation.
func WithStructuralValidation() Option {
	return func(o *Options) { o.structuralValidation = true }
}

type Option func(o *Options)

func (o *Options) apply(opts []Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func newOptions(options []Option) *Options {
	ret := &Options{}
	ret.apply(options)
	return ret
}

func WithErrorHandler(fn func(err error, cur *parsly.Cursor, destNode interface{}) error) Option {
	return func(o *Options) {
		o.onError = fn
	}
}

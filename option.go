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

// WithErrorHandler extends syntax at parser-owned recovery points. For an
// unknown operand, destNode is *node.Node: a successful handler must assign a
// node and advance the cursor through exactly that operand. Nested argument
// and subquery cursors inherit the handler; unhandled errors must be returned.
func WithErrorHandler(fn func(err error, cur *parsly.Cursor, destNode interface{}) error) Option {
	return func(o *Options) {
		o.onError = fn
	}
}

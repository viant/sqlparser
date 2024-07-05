package insert

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

type Value struct {
	Expr     node.Node
	Comments string
	Raw      string
	node.Span
	Meta interface{}
}

func (v *Value) IsPlaceholder() bool {
	return v.Raw == "?"
}
func (v *Value) Interface() (interface{}, error) {
	return expr.NewValue(v.Raw)
}

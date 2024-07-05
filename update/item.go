package update

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

// Item represents an update item
type Item struct {
	node.Span
	Column   node.Node
	Expr     node.Node
	Comments string
	Raw      string
	Meta     interface{}
}

func (v *Item) IsPlaceholder() bool {
	return v.Raw == "?"
}
func (v *Item) Interface() (interface{}, error) {
	return expr.NewValue(v.Raw)
}

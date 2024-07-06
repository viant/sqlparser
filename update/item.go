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
	_, ok := v.Expr.(*expr.Placeholder)
	return ok
}
func (v *Item) IsExpr() bool {
	_, ok := v.Expr.(*expr.Binary)
	return ok
}

func (v *Item) Value() (*expr.Value, error) {
	return expr.NewValue(v.Raw)
}

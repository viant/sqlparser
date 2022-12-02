package insert

import "github.com/viant/sqlparser/node"

type Value struct {
	Expr     node.Node
	Comments string
	Raw      string
	node.Span
	Meta interface{}
}

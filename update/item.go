package update

import "github.com/viant/sqlx/metadata/ast/node"

type Item struct {
	node.Span
	Column   node.Node
	Expr     node.Node
	Comments string
	Raw      string
	Meta     interface{}
}

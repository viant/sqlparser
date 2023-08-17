package query

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

// From represetns from
type From struct {
	expr.Raw
	Alias    string
	X        node.Node
	Comments string
}

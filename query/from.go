package query

import (
	"github.com/viant/sqlparser/node"
)

//From represetns from
type From struct {
	Alias    string
	X        node.Node
	Comments string
}

package query

import (
	"github.com/viant/sqlparser/node"
)

type From struct {
	Alias    string
	X        node.Node
	Comments string
}

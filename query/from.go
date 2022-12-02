package query

import (
	"github.com/viant/sqlx/metadata/ast/node"
)

type From struct {
	Alias    string
	X        node.Node
	Comments string
}

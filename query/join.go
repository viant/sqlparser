package query

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

type Join struct {
	Kind     string
	Alias    string
	Raw      string
	With     node.Node
	On       *expr.Qualify
	Comments string
}

func NewJoin(raw string) *Join {
	return &Join{Raw: raw}
}

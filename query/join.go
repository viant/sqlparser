package query

import (
	"github.com/viant/sqlx/metadata/ast/expr"
	"github.com/viant/sqlx/metadata/ast/node"
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

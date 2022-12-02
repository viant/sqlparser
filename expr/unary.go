package expr

import "github.com/viant/sqlx/metadata/ast/node"

type Unary struct {
	Op string
	X  node.Node
}

func NewUnary(op string) *Unary {
	return &Unary{Op: op}
}

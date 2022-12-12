package expr

import "github.com/viant/sqlparser/node"

//Unary represents  unary expr
type Unary struct {
	Op string
	X  node.Node
}

//NewUnary crates an unary expr
func NewUnary(op string) *Unary {
	return &Unary{Op: op}
}

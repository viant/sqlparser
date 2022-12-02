package expr

import "github.com/viant/sqlx/metadata/ast/node"

type Binary struct {
	X, Y node.Node
	Op   string
}

func NewBinary(x node.Node) *Binary {
	return &Binary{X: x}
}

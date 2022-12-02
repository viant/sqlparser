package expr

import "github.com/viant/sqlparser/node"

type Binary struct {
	X, Y node.Node
	Op   string
}

func NewBinary(x node.Node) *Binary {
	return &Binary{X: x}
}

package expr

import "github.com/viant/sqlparser/node"

type Qualify struct {
	X node.Node
}

func NewQualify() *Qualify {
	return &Qualify{}
}

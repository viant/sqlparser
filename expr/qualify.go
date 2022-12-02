package expr

import "github.com/viant/sqlx/metadata/ast/node"

type Qualify struct {
	X node.Node
}

func NewQualify() *Qualify {
	return &Qualify{}
}

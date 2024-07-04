package expr

import "github.com/viant/sqlparser/node"

// Qualify represents qualify node
type Qualify struct {
	X node.Node
}

// NewQualify returns qualify node
func NewQualify() *Qualify {
	return &Qualify{}
}

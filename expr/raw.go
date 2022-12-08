package expr

import "github.com/viant/sqlparser/node"

type Raw struct {
	Raw string
	X   node.Node
}

func NewRaw(raw string) *Raw {
	return &Raw{Raw: raw}
}

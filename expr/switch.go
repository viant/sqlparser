package expr

import "github.com/viant/sqlparser/node"

type (
	Switch struct {
		Raw string
		Ident
		Cases []*Case
	}

	Case struct {
		X Qualify
		Y node.Node
	}
)

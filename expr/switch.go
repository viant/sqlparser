package expr

import "github.com/viant/sqlparser/node"

type (
	//Switch represetns a switch expr
	Switch struct {
		Raw string
		Ident
		Cases []*Case
	}

	//Case represents a switch case
	Case struct {
		X Qualify
		Y node.Node
	}
)

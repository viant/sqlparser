package expr

import "github.com/viant/sqlx/metadata/ast/node"

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

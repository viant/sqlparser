package expr

import "github.com/viant/sqlparser/node"

type (
	//Switch represetns a switch expr
	Switch struct {
		Raw string
		Ident
		Cases []*Case
	}

	// Case represents a WHEN condition (X) and THEN result (Y).
	// A first entry with only X holds a simple CASE selector; an entry with
	// only Y holds ELSE. This keeps all expressions traversable without
	// changing the public Switch or Case field layout.
	Case struct {
		X Qualify
		Y node.Node
	}
)

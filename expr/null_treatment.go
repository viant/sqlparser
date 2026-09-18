package expr

import "github.com/viant/sqlparser/node"

// NullTreatment applies IGNORE NULLS or RESPECT NULLS to an aggregate argument.
type NullTreatment struct {
	X    node.Node
	Mode string
}

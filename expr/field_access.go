package expr

import "github.com/viant/sqlparser/node"

// FieldAccess selects a field of a computed value, such as an array element.
type FieldAccess struct {
	X    node.Node
	Name string
}

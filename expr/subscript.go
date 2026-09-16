package expr

import "github.com/viant/sqlparser/node"

// Subscript accesses an element of X using an index expression.
type Subscript struct {
	X     node.Node
	Index node.Node
}

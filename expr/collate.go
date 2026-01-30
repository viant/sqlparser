package expr

import "github.com/viant/sqlparser/node"

// Collate represents collate expression.
type Collate struct {
	X         node.Node
	Collation string
}

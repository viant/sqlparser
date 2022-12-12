package update

import "github.com/viant/sqlparser/node"

//Target represents a target
type Target struct {
	X        node.Node
	Comments string
}

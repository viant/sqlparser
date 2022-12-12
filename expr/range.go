package expr

import "github.com/viant/sqlparser/node"

//Range represents a range expr
type Range struct {
	Min node.Node
	Max node.Node
}

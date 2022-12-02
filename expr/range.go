package expr

import "github.com/viant/sqlparser/node"

type Range struct {
	Min node.Node
	Max node.Node
}

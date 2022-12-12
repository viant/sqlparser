package expr

import "github.com/viant/sqlparser/node"

//Call represents a call
type Call struct {
	X    node.Node
	Args []node.Node
	Raw  string
}

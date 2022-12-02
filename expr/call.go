package expr

import "github.com/viant/sqlparser/node"

type Call struct {
	X    node.Node
	Args []node.Node
	Raw  string
}

package del

import "github.com/viant/sqlparser/node"

type Target struct {
	X        node.Node
	Comments string
	Alias    string
}

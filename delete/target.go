package del

import "github.com/viant/sqlparser/node"

//Target represents deletion target
type Target struct {
	X        node.Node
	Comments string
	Alias    string
}

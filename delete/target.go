package del

import "github.com/viant/sqlx/metadata/ast/node"

type Target struct {
	X        node.Node
	Comments string
	Alias    string
}

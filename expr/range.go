package expr

import "github.com/viant/sqlx/metadata/ast/node"

type Range struct {
	Min node.Node
	Max node.Node
}

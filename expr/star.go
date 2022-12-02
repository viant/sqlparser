package expr

import "github.com/viant/sqlx/metadata/ast/node"

type Star struct {
	X        node.Node
	Except   []string
	Comments string
}

func NewStar(x node.Node, comments string) *Star {
	return &Star{X: x, Comments: comments}
}

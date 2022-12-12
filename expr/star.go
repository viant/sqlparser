package expr

import "github.com/viant/sqlparser/node"

//Star represents a star expr
type Star struct {
	X        node.Node
	Except   []string
	Comments string
}

//NewStar returns a start expr
func NewStar(x node.Node, comments string) *Star {
	return &Star{X: x, Comments: comments}
}

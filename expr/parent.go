package expr

import "github.com/viant/sqlparser/node"

type Parenthesis struct {
	Raw string
	X   node.Node
}

func NewParenthesis(raw string) *Parenthesis {
	return &Parenthesis{Raw: raw}
}

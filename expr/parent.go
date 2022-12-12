package expr

import "github.com/viant/sqlparser/node"

//Parenthesis represents parenthesis expr
type Parenthesis struct {
	Raw string
	X   node.Node
}

//NewParenthesis returns a parenthesis expr
func NewParenthesis(raw string) *Parenthesis {
	return &Parenthesis{Raw: raw}
}

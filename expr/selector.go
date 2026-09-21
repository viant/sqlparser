package expr

import (
	"strings"

	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/source"
)

// Selector represent identifier selector
type Selector struct {
	Name       string
	Expression string
	X          node.Node
}

// NewSelector returns
func NewSelector(name string) node.Node {
	var expr = ""
	if exprPos := source.FindCodeToken(name, "[", 0); exprPos != -1 {
		expr = name[exprPos:]
		exprEnd := strings.Index(expr, "]")
		if exprEnd != -1 {
			expr = expr[:exprEnd+1]
		}
		name = name[:exprPos] + name[exprPos+len(expr):]
		expr = expr[1 : len(expr)-1]
	}
	// Dots and brackets inside quoted segments are identifier text.
	part := source.FindCodeToken(name, ".", 0)
	if part == -1 {
		if expr != "" {
			return &Selector{Name: name, Expression: expr}
		}
		return &Ident{Name: name}
	}
	return &Selector{Name: name[:part], X: NewSelector(name[part+1:]), Expression: expr}
}

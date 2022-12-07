package expr

import "github.com/viant/sqlparser/node"

type Ident struct {
	Name string
}

//Identity returns ident or selector expr
func Identity(node node.Node) node.Node {
	switch actual := node.(type) {
	case *Ident:
		return actual
	case *Selector:
		return actual
	}
	return nil
}

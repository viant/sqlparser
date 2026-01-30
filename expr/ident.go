package expr

import "github.com/viant/sqlparser/node"

// Ident represent an identifier
type Ident struct {
	Name string
}

// Identity returns ident or selector expr
func Identity(node node.Node) node.Node {
	switch actual := node.(type) {
	case *Ident:
		return actual
	case *Selector:
		return actual
	case *Collate:
		return Identity(actual.X)
	}
	return nil
}

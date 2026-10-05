package expr

import "github.com/viant/sqlparser/node"

// Window is a function call with an inline OVER specification.
type Window struct {
	X           node.Node
	PartitionBy []node.Node
	OrderBy     []*WindowOrder
	Frame       *WindowFrame
}

// WindowOrder is an expression and its optional ASC/DESC direction.
type WindowOrder struct {
	X         node.Node
	Direction string
}

// WindowFrame represents ROWS or RANGE, with an optional BETWEEN end bound.
type WindowFrame struct {
	Unit       string
	Start, End *WindowBound
}

// WindowBound is CURRENT ROW, UNBOUNDED PRECEDING/FOLLOWING, or an offset bound.
type WindowBound struct {
	X    node.Node
	Kind string
}

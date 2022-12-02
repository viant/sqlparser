package query

import (
	"github.com/viant/sqlparser/node"
)

type Item struct {
	Expr      node.Node
	Alias     string
	Comments  string
	DataType  string
	Raw       string
	Meta      interface{}
	Direction string
}

func NewItem(expr node.Node) *Item {
	return &Item{Expr: expr}
}

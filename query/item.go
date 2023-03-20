package query

import (
	"github.com/viant/sqlparser/node"
)

//Item represents an inte,
type Item struct {
	Expr      node.Node
	Alias     string
	Comments  string
	DataType  string
	Raw       string
	Meta      interface{}
	Tag       string
	Direction string
}

//NewItem returns an item
func NewItem(expr node.Node) *Item {
	return &Item{Expr: expr}
}

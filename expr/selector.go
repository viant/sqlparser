package expr

import (
	"github.com/viant/sqlx/metadata/ast/node"
	"strings"
)

type Selector struct {
	Name string
	X    node.Node
}

func NewSelector(name string) node.Node {
	part := strings.Index(name, ".")
	if part == -1 {
		return &Ident{Name: name}
	}
	return &Selector{Name: name[:part], X: NewSelector(name[part+1:])}
}

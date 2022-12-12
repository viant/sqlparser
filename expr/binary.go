package expr

import (
	"github.com/viant/sqlparser/node"
)

//Binary represents binary expr
type Binary struct {
	X, Y node.Node
	Op   string
}

//HasPlaceholder returns true if x or y operand is placeholder
func (b *Binary) HasPlaceholder() bool {
	if _, ok := b.X.(*Placeholder); ok {
		return ok
	}
	_, ok := b.Y.(*Placeholder)
	return ok
}

//Parenthesis returns parenthesis
func (b *Binary) Parenthesis() *Parenthesis {
	if p, ok := b.X.(*Parenthesis); ok {
		return p
	}
	p, _ := b.Y.(*Parenthesis)
	return p
}

//HasIdentifier returns true if x or y opperand is identity
func (b *Binary) HasIdentifier() bool {
	return b.Identifier() != nil
}

//Identifier returns an identifier node or nil
func (b *Binary) Identifier() node.Node {
	if x := Identity(b.X); x != nil {
		return x
	}
	return Identity(b.Y)
}

//NewBinary returns a binary expr
func NewBinary(x node.Node) *Binary {
	return &Binary{X: x}
}

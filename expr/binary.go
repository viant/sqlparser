package expr

import (
	"github.com/viant/sqlparser/node"
)

// Binary represents binary expr
type Binary struct {
	X, Y node.Node
	Op   string
}

func (b *Binary) Normalize() *Binary {
	normalized := *b
	switch b.Op[0] {
	case 'A', 'O', 'a', 'o':
	default:

		if bin, ok := b.Y.(*Binary); ok && Identity(b.X) != nil {
			xBin := &Binary{X: b.X, Y: bin.X, Op: b.Op}
			normalized.X = xBin
			normalized.Op = bin.Op
			normalized.Y = bin.Y
		}
	}
	return &normalized
}

func (b *Binary) Walk(fn func(ident node.Node, values *Values, operator, parentOperator string) error) error {
	b = b.Normalize()
	switch b.Op[0] {
	case 'A', 'O', 'a', 'o':
		if x, ok := b.X.(*Binary); ok {
			if err := x.walk(fn, b.Op); err != nil {
				return err
			}
		}
		if y, ok := b.Y.(*Binary); ok {
			if err := y.walk(fn, b.Op); err != nil {
				return err
			}
		}
		return nil
	}
	return b.walk(fn, "")

}

func (b *Binary) walk(fn func(ident node.Node, values *Values, operator, parentOperator string) error, operator string) error {
	switch b.Op[0] {
	case 'A', 'O', 'a', 'o':
		if x, ok := b.X.(*Binary); ok {
			if err := x.walk(fn, b.Op); err != nil {
				return err
			}
		}
		if y, ok := b.Y.(*Binary); ok {
			if err := y.walk(fn, b.Op); err != nil {
				return err
			}
		}
		return nil
	}

	sel, values, err := b.Predicate()
	if err != nil {
		return err
	}
	err = fn(sel, values, b.Op, operator)
	if err != nil {
		return err
	}
	if binY, ok := b.Y.(*Binary); ok {
		if nested, ok := binY.Y.(*Binary); ok {
			if err = nested.walk(fn, b.Op); err != nil {
				return err
			}
		}

	}
	return nil
}

// Predicate binary predicate or nil
func (b *Binary) Predicate() (node.Node, *Values, error) {
	switch b.Op[0] {
	case 'A', 'O':
		return nil, nil, nil
	}
	identifier := b.Identifier()
	if identifier == nil {
		return nil, nil, nil
	}
	values, err := b.Values()
	if err != nil {
		return nil, nil, err
	}
	return identifier, values, nil
}

// Placeholder returns  placeholder
func (b *Binary) Placeholder() *Placeholder {
	r, ok := b.X.(*Placeholder)
	if ok {
		return r
	}
	r, ok = b.Y.(*Placeholder)
	return r
}

// HasPlaceholder returns true if x or y operand is placeholder
func (b *Binary) HasPlaceholder() bool {
	if _, ok := b.X.(*Placeholder); ok {
		return ok
	}
	_, ok := b.Y.(*Placeholder)
	return ok
}

// Parenthesis returns parenthesis
func (b *Binary) Parenthesis() *Parenthesis {
	if p, ok := b.X.(*Parenthesis); ok {
		return p
	}
	p, _ := b.Y.(*Parenthesis)
	return p
}

// HasIdentifier returns true if x or y opperand is identity
func (b *Binary) HasIdentifier() bool {
	return b.Identifier() != nil
}

// Identifier returns an identifier node or nil
func (b *Binary) Identifier() node.Node {
	if x := Identity(b.X); x != nil {
		return x
	}
	return Identity(b.Y)
}

func (b *Binary) Ident() *Ident {
	if x, ok := b.X.(*Ident); ok {
		return x
	}
	if y, ok := b.Y.(*Ident); ok {
		return y
	}
	return nil
}

func (b *Binary) SelectorIdent(root string) *Ident {
	if x, ok := b.X.(*Selector); ok {
		if x.Name == root {
			return x.X.(*Ident)
		}
		return nil
	}
	if y, ok := b.Y.(*Selector); ok {
		if y.Name == root {
			return y.X.(*Ident)
		}
	}
	return nil
}

// Values returns expression values
func (b *Binary) Values() (*Values, error) {
	if x := Identity(b.X); x == nil {
		return NewValues(b.X)
	}
	return NewValues(b.Y)
}

// NewBinary returns a binary expr
func NewBinary(x node.Node) *Binary {
	return &Binary{X: x}
}

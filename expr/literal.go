package expr

type Literal struct {
	Value string
	Kind  string
}

func NewIntLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "int"}
}

func NewStringLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "string"}
}

func NewBoolLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "bool"}
}

func NewNullLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "null"}
}

func NewNumericLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "numeric"}
}

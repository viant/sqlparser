package expr

//Literal represents a literal
type Literal struct {
	Value string
	Kind  string
}

//NewIntLiteral returns an int literal
func NewIntLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "int"}
}

//NewStringLiteral returns string literal
func NewStringLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "string"}
}

//NewBoolLiteral returns bool literal
func NewBoolLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "bool"}
}

//NewNullLiteral returns null literal
func NewNullLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "null"}
}

//NewNumericLiteral returns numeric literal
func NewNumericLiteral(v string) *Literal {
	return &Literal{Value: v, Kind: "numeric"}
}

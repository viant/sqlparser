package expr

type Parenthesis struct {
	Raw string
}

func NewParenthesis(raw string) *Parenthesis {
	return &Parenthesis{Raw: raw}
}

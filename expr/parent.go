package expr

type Parenthesis struct {
	Raw string
}

func NewParenthesis(raww string) *Parenthesis {
	return &Parenthesis{Raw: raww}
}

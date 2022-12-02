package expr

type Placeholder struct {
	Name     string
	Comments string
}

func NewPlaceholder(name string) *Placeholder {
	return &Placeholder{Name: name}
}

package expr

//Placeholder represetns a placeholder
type Placeholder struct {
	Name     string
	Comments string
}

//NewPlaceholder returns a placeholder
func NewPlaceholder(name string) *Placeholder {
	return &Placeholder{Name: name}
}

package expr

type Raw struct {
	Raw string
}

func NewRaw(raw string) *Raw {
	return &Raw{Raw: raw}
}

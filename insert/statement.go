package insert

type Statement struct {
	Target  Target
	Columns []string
	Values  []*Value
}

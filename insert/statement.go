package insert

//Statement represetns an insert stmt
type Statement struct {
	Target  Target
	Columns []string
	Values  []*Value
}

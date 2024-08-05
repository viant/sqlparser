package index

// Drop represents a drop
type Drop struct {
	IfExists bool
	Name     string
	Schema   string
	Table    string
}

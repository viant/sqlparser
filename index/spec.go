package index

type (
	//Spec represents index specification
	Spec struct {
		Name    string
		Schema  string
		Table   string
		SQL     string
		Type    string
		Storage string
		Columns []*ColumnSpec
	}

	ColumnSpec struct {
		Name string
		Type string
	}
)

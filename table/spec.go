package table

import "github.com/viant/sqlparser/column"

type (
	//Spec represents table specification
	Spec struct {
		Name    string
		Columns []*column.Spec
	}
)

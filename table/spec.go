package table

import (
	"github.com/viant/sqlparser/column"
)

type (
	//Column represents table specification
	Spec struct {
		Name    string
		SQL     string
		Columns []*column.Spec
	}
)

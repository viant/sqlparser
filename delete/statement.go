package delete

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

// Statement represents delete statement
type Statement struct {
	Target  Target
	Items   []*Item
	Joins   []*query.Join
	Qualify *expr.Qualify
}

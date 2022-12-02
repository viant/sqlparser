package del

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

type Statement struct {
	Target  Target
	Items   []*Item
	Joins   []*query.Join
	Qualify *expr.Qualify
}

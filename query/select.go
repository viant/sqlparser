package query

import (
	"github.com/viant/sqlx/metadata/ast/expr"
)

type Select struct {
	List    List
	From    From
	Joins   []*Join
	Qualify *expr.Qualify
	GroupBy []string
	Having  *expr.Qualify
	OrderBy List
	Window  *expr.Raw
	Limit   *expr.Literal
	Offset  *expr.Literal
	Kind    string
}

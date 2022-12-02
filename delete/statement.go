package del

import (
	"github.com/viant/sqlx/metadata/ast/expr"
	"github.com/viant/sqlx/metadata/ast/query"
)

type Statement struct {
	Target  Target
	Items   []*Item
	Joins   []*query.Join
	Qualify *expr.Qualify
}

package update

import (
	"github.com/viant/sqlx/metadata/ast/expr"
)

type Statement struct {
	Target  Target
	Set     []*Item
	Qualify *expr.Qualify
}

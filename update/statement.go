package update

import (
	"github.com/viant/sqlparser/expr"
)

type Statement struct {
	Target  Target
	Set     []*Item
	Qualify *expr.Qualify
}

package update

import (
	"github.com/viant/sqlparser/expr"
)

//Statement represetns an update statment
type Statement struct {
	Target  Target
	Set     []*Item
	Qualify *expr.Qualify
}

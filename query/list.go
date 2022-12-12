package query

import "github.com/viant/sqlparser/expr"

//List represents a list
type List []*Item

//Append appends to the list
func (l *List) Append(item *Item) {
	*l = append(*l, item)
}

//IsStarExpr returns true if start expr
func (l List) IsStarExpr() bool {
	if len(l) == 1 {
		switch actual := l[0].Expr.(type) {
		case *expr.Star:
			return true
		case *expr.Selector:
			if _, ok := actual.X.(*expr.Star); ok {
				return ok
			}
		}
	}
	return false
}

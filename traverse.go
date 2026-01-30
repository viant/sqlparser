package sqlparser

import (
	"fmt"
	"github.com/viant/sqlparser/column"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/table"
	"github.com/viant/sqlparser/update"
)

// Traverse traverse node
func Traverse(n node.Node, visitor func(n node.Node) bool) {
	traverse(n, visitor)
}

func traverse(n node.Node, visitor func(n node.Node) bool) bool {
	if n == nil {
		return false
	}

	if !visitor(n) {
		return false
	}
	switch actual := n.(type) {
	case string:
	case *query.Select:
		traverse(actual.List, visitor)
		traverse(&actual.From, visitor)

		if len(actual.Joins) > 0 {
			for _, join := range actual.Joins {
				traverse(join, visitor)
			}
		}
		if actual.Qualify != nil {
			traverse(actual.Qualify.X, visitor)
		}
		if len(actual.GroupBy) > 0 {
			for _, item := range actual.GroupBy {
				traverse(item, visitor)
			}
		}
		if actual.Having != nil {
			traverse(actual.Having, visitor)
		}

		if len(actual.OrderBy) > 0 {
			for _, item := range actual.OrderBy {
				traverse(item, visitor)
			}
		}
		if union := actual.Union; union != nil {
			traverse(union.X, visitor)
		}

	case *query.Join:
		traverse(actual.With, visitor)
		traverse(actual.On, visitor)
	case *expr.Qualify:
		traverse(actual.X, visitor)
	case *expr.Literal:
	case query.List:
		listSize := len(actual)
		if listSize == 0 {
			return true
		}
		traverse(actual[0], visitor)
		for i := 1; i < listSize; i++ {
			traverse(actual[i], visitor)
		}

	case *expr.Star:
		traverse(actual.X, visitor)
	case *expr.Raw:
		traverse(actual.Raw, visitor)
	case *query.From:
		if actual.X == nil {
			return true
		}
		traverse(actual.X, visitor)
	case *expr.Placeholder:
	case *expr.Unary:
		traverse(actual.X, visitor)
	case *expr.Parenthesis:
		traverse(actual.X, visitor)
	case *query.Item:
		traverse(actual.Expr, visitor)
	case *expr.Binary:
		if !traverse(actual.X, visitor) {
			return false
		}
		if actual.Y != nil {
			if !traverse(actual.Y, visitor) {
				return false
			}
		}
	case expr.Raw:
		if !traverse(actual.X, visitor) {
			return false
		}
	case *expr.Ident:
	case *expr.Call:
		traverse(actual.X, visitor)
		for _, arg := range actual.Args {
			traverse(arg, visitor)
		}
	case *expr.Collate:
		traverse(actual.X, visitor)
	case *expr.Range:
		traverse(actual.Min, visitor)
		traverse(actual.Max, visitor)
	case *expr.Selector:

		traverse(actual.X, visitor)
	case *update.Item:
		traverse(actual.Column, visitor)
		traverse(actual.Expr, visitor)
	case *update.Statement:
		traverse(actual.Target.X, visitor)
		for i := range actual.Set {
			traverse(actual.Set[i], visitor)
		}
		if actual.Qualify != nil {
			traverse(actual.Qualify, visitor)
		}
	case *insert.Statement:
		traverse(actual.Target.X, visitor)
		for _, value := range actual.Values {
			traverse(value.Expr, visitor)
		}
	case *del.Statement:
		for _, item := range actual.Items {
			traverse(item, visitor)
		}
		traverse(actual.Target, visitor)
		for _, join := range actual.Joins {
			traverse(join, visitor)
		}
		if actual.Qualify != nil {
			traverse(actual.Qualify, visitor)
		}
	case del.Target:
		traverse(actual.X, visitor)
	case *del.Item:
	case *table.Create:
		for _, col := range actual.Columns {
			traverse(col, visitor)
		}
	case *column.Spec:
	default:
		panic(fmt.Sprintf("%T unsupported", n))
	}

	return true
}

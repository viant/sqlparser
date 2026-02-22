package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

// StripCollate removes COLLATE expressions from SQL while preserving structure.
func StripCollate(SQL string) (string, error) {
	parsed, err := ParseQuery(SQL)
	if err != nil {
		return "", err
	}
	stripCollateSelect(parsed)
	return Stringify(parsed), nil
}

func stripCollateSelect(sel *query.Select) {
	if sel == nil {
		return
	}
	for i := range sel.List {
		sel.List[i].Expr = stripCollateNode(sel.List[i].Expr)
	}
	if sel.From.X != nil {
		sel.From.X = stripCollateNode(sel.From.X)
	}
	for _, join := range sel.Joins {
		join.With = stripCollateNode(join.With)
		join.On = stripQualify(join.On)
	}
	if sel.Qualify != nil {
		sel.Qualify.X = stripCollateNode(sel.Qualify.X)
	}
	for i := range sel.GroupBy {
		sel.GroupBy[i].Expr = stripCollateNode(sel.GroupBy[i].Expr)
	}
	if sel.Having != nil {
		sel.Having.X = stripCollateNode(sel.Having.X)
	}
	for i := range sel.OrderBy {
		sel.OrderBy[i].Expr = stripCollateNode(sel.OrderBy[i].Expr)
	}
	if sel.Union != nil {
		stripCollateSelect(sel.Union.X)
	}
	for _, withSel := range sel.WithSelects {
		stripCollateSelect(withSel.X)
	}
}

func stripCollateNode(n node.Node) node.Node {
	switch actual := n.(type) {
	case *expr.Collate:
		return stripCollateNode(actual.X)
	case *expr.Binary:
		actual.X = stripCollateNode(actual.X)
		if actual.Y != nil {
			actual.Y = stripCollateNode(actual.Y)
		}
		return actual
	case *expr.Parenthesis:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *expr.Unary:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *expr.Call:
		actual.X = stripCollateNode(actual.X)
		for i := range actual.Args {
			actual.Args[i] = stripCollateNode(actual.Args[i])
		}
		return actual
	case *expr.Star:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *expr.Selector:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *expr.Qualify:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *expr.Range:
		actual.Min = stripCollateNode(actual.Min)
		actual.Max = stripCollateNode(actual.Max)
		return actual
	case *expr.Switch:
		for _, c := range actual.Cases {
			c.X.X = stripCollateNode(c.X.X)
			c.Y = stripCollateNode(c.Y)
		}
		return actual
	case *expr.Raw:
		if actual.X != nil {
			actual.X = stripCollateNode(actual.X)
		}
		return actual
	case *query.Select:
		stripCollateSelect(actual)
		return actual
	case *query.Item:
		actual.Expr = stripCollateNode(actual.Expr)
		return actual
	case query.List:
		for i := range actual {
			actual[i].Expr = stripCollateNode(actual[i].Expr)
		}
		return actual
	case *query.From:
		actual.X = stripCollateNode(actual.X)
		return actual
	case *query.Join:
		actual.With = stripCollateNode(actual.With)
		actual.On = stripQualify(actual.On)
		return actual
	case *query.Union:
		stripCollateSelect(actual.X)
		return actual
	}
	return n
}

func stripQualify(input *expr.Qualify) *expr.Qualify {
	if input == nil {
		return nil
	}
	if stripped, ok := stripCollateNode(input).(*expr.Qualify); ok {
		return stripped
	}
	return input
}

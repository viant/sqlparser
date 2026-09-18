package sqlparser

import (
	"fmt"
	"strings"

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
	if err = stripCollateSelect(parsed); err != nil {
		return "", err
	}
	return Stringify(parsed), nil
}

func stripCollateSelect(sel *query.Select) error {
	if sel == nil {
		return nil
	}
	for i := range sel.List {
		stripped, err := stripCollateNode(sel.List[i].Expr)
		if err != nil {
			return err
		}
		sel.List[i].Expr = stripped
	}
	if sel.From.X != nil {
		stripped, err := stripCollateNode(sel.From.X)
		if err != nil {
			return err
		}
		sel.From.X = stripped
	}
	for _, join := range sel.Joins {
		stripped, err := stripCollateNode(join.With)
		if err != nil {
			return err
		}
		join.With = stripped
		join.On, err = stripQualify(join.On)
		if err != nil {
			return err
		}
	}
	if sel.Qualify != nil {
		stripped, err := stripCollateNode(sel.Qualify.X)
		if err != nil {
			return err
		}
		sel.Qualify.X = stripped
	}
	for i := range sel.GroupBy {
		stripped, err := stripCollateNode(sel.GroupBy[i].Expr)
		if err != nil {
			return err
		}
		sel.GroupBy[i].Expr = stripped
	}
	if sel.Having != nil {
		stripped, err := stripCollateNode(sel.Having.X)
		if err != nil {
			return err
		}
		sel.Having.X = stripped
	}
	for i := range sel.OrderBy {
		stripped, err := stripCollateNode(sel.OrderBy[i].Expr)
		if err != nil {
			return err
		}
		sel.OrderBy[i].Expr = stripped
	}
	if sel.Union != nil {
		if err := stripCollateSelect(sel.Union.X); err != nil {
			return err
		}
	}
	for _, withSel := range sel.WithSelects {
		if err := stripCollateSelect(withSel.X); err != nil {
			return err
		}
		if withSel.X != nil {
			withSel.Raw = "(" + (Stringifier{PreserveWindow: true}).String(withSel.X) + ")"
		}
	}
	return nil
}

func stripCollateNode(n node.Node) (node.Node, error) {
	switch actual := n.(type) {
	case *expr.Collate:
		return stripCollateNode(actual.X)
	case *expr.FieldAccess:
		var err error
		actual.X, err = stripCollateNode(actual.X)
		return actual, err
	case *expr.NullTreatment:
		var err error
		actual.X, err = stripCollateNode(actual.X)
		return actual, err
	case *expr.Subscript:
		var err error
		actual.X, err = stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.Index, err = stripCollateNode(actual.Index)
		return actual, err
	case *expr.Binary:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		if actual.Y != nil {
			stripped, err = stripCollateNode(actual.Y)
			if err != nil {
				return nil, err
			}
			actual.Y = stripped
		}
		return actual, nil
	case *expr.Parenthesis:
		if !hasCollate(actual.X) {
			return actual, nil
		}
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		if actual.X != nil {
			actual.Raw = "(" + (Stringifier{PreserveWindow: true}).String(actual.X) + ")"
		}
		return actual, nil
	case *expr.Unary:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		return actual, nil
	case *expr.Call:
		if !hasCollate(actual) {
			return actual, nil
		}
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		for i := range actual.Args {
			stripped, err = stripCollateNode(actual.Args[i])
			if err != nil {
				return nil, err
			}
			actual.Args[i] = stripped
		}
		if len(actual.Args) > 0 {
			args := make([]string, 0, len(actual.Args))
			for _, arg := range actual.Args {
				// Pagination inside a query argument determines its value.
				args = append(args, (Stringifier{PreserveWindow: true}).String(arg))
			}
			actual.Raw = "(" + strings.Join(args, ", ") + ")"
		}
		return actual, nil
	case *expr.Star:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		return actual, nil
	case *expr.Selector:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		return actual, nil
	case *expr.Qualify:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		return actual, nil
	case *expr.Range:
		stripped, err := stripCollateNode(actual.Min)
		if err != nil {
			return nil, err
		}
		actual.Min = stripped
		stripped, err = stripCollateNode(actual.Max)
		if err != nil {
			return nil, err
		}
		actual.Max = stripped
		return actual, nil
	case *expr.Switch:
		if !hasCollate(actual) {
			return actual, nil
		}
		for _, c := range actual.Cases {
			if c == nil {
				continue
			}
			stripped, err := stripCollateNode(c.X.X)
			if err != nil {
				return nil, err
			}
			c.X.X = stripped
			stripped, err = stripCollateNode(c.Y)
			if err != nil {
				return nil, err
			}
			c.Y = stripped
		}
		actual.Raw = ""
		actual.Raw = (Stringifier{PreserveWindow: true}).String(actual)
		return actual, nil
	case []node.Node:
		for i := range actual {
			stripped, err := stripCollateNode(actual[i])
			if err != nil {
				return nil, err
			}
			actual[i] = stripped
		}
		return actual, nil
	case *expr.Raw:
		hadCollate := hasCollate(actual.X)
		if actual.X != nil {
			stripped, err := stripCollateNode(actual.X)
			if err != nil {
				return nil, err
			}
			actual.X = stripped
		}
		if hadCollate {
			if !rawParsedFaithfully(actual) {
				return nil, fmt.Errorf("cannot strip COLLATE from raw subquery with opaque template fragments")
			}
			actual.Raw = "(" + (Stringifier{PreserveWindow: true}).String(actual.X) + ")"
		}
		return actual, nil
	case *query.Select:
		return actual, stripCollateSelect(actual)
	case *query.Item:
		stripped, err := stripCollateNode(actual.Expr)
		if err != nil {
			return nil, err
		}
		actual.Expr = stripped
		return actual, nil
	case query.List:
		for i := range actual {
			stripped, err := stripCollateNode(actual[i].Expr)
			if err != nil {
				return nil, err
			}
			actual[i].Expr = stripped
		}
		return actual, nil
	case *query.From:
		stripped, err := stripCollateNode(actual.X)
		if err != nil {
			return nil, err
		}
		actual.X = stripped
		return actual, nil
	case *query.Join:
		stripped, err := stripCollateNode(actual.With)
		if err != nil {
			return nil, err
		}
		actual.With = stripped
		actual.On, err = stripQualify(actual.On)
		if err != nil {
			return nil, err
		}
		return actual, nil
	case *query.Union:
		return actual, stripCollateSelect(actual.X)
	}
	return n, nil
}

func stripQualify(input *expr.Qualify) (*expr.Qualify, error) {
	if input == nil {
		return nil, nil
	}
	stripped, err := stripCollateNode(input)
	if err != nil {
		return nil, err
	}
	if qualify, ok := stripped.(*expr.Qualify); ok {
		return qualify, nil
	}
	return input, nil
}

func rawParsedFaithfully(raw *expr.Raw) bool {
	if raw == nil {
		return false
	}
	rawExpr := trimEnclosure(raw.Raw)
	return strings.TrimSpace(rawExpr) == strings.TrimSpace(stripTemplateBuiltins(rawExpr))
}

func hasCollate(n node.Node) bool {
	switch actual := n.(type) {
	case nil:
		return false
	case *expr.Collate:
		return true
	case *expr.Subscript:
		return hasCollate(actual.X) || hasCollate(actual.Index)
	case *expr.FieldAccess:
		return hasCollate(actual.X)
	case *expr.NullTreatment:
		return hasCollate(actual.X)
	case *expr.Binary:
		return hasCollate(actual.X) || hasCollate(actual.Y)
	case *expr.Parenthesis:
		return hasCollate(actual.X)
	case *expr.Unary:
		return hasCollate(actual.X)
	case *expr.Call:
		if hasCollate(actual.X) {
			return true
		}
		for _, arg := range actual.Args {
			if hasCollate(arg) {
				return true
			}
		}
	case *expr.Star:
		return hasCollate(actual.X)
	case *expr.Selector:
		return hasCollate(actual.X)
	case *expr.Qualify:
		return hasCollate(actual.X)
	case *expr.Range:
		return hasCollate(actual.Min) || hasCollate(actual.Max)
	case *expr.Switch:
		for _, c := range actual.Cases {
			if c != nil && (hasCollate(c.X.X) || hasCollate(c.Y)) {
				return true
			}
		}
	case []node.Node:
		for _, item := range actual {
			if hasCollate(item) {
				return true
			}
		}
	case *expr.Raw:
		return hasCollate(actual.X)
	case *query.Select:
		for _, item := range actual.List {
			if hasCollate(item.Expr) {
				return true
			}
		}
		if hasCollate(actual.From.X) {
			return true
		}
		for _, join := range actual.Joins {
			if hasCollate(join.With) || join.On != nil && hasCollate(join.On.X) {
				return true
			}
		}
		if actual.Qualify != nil && hasCollate(actual.Qualify.X) {
			return true
		}
		for _, item := range actual.GroupBy {
			if hasCollate(item.Expr) {
				return true
			}
		}
		if actual.Having != nil && hasCollate(actual.Having.X) {
			return true
		}
		for _, item := range actual.OrderBy {
			if hasCollate(item.Expr) {
				return true
			}
		}
		if actual.Union != nil && hasCollate(actual.Union.X) {
			return true
		}
		for _, withSel := range actual.WithSelects {
			if hasCollate(withSel.X) {
				return true
			}
		}
		return false
	case *query.Item:
		return hasCollate(actual.Expr)
	case query.List:
		for _, item := range actual {
			if hasCollate(item.Expr) {
				return true
			}
		}
	case *query.From:
		return hasCollate(actual.X)
	case *query.Join:
		return hasCollate(actual.With) || actual.On != nil && hasCollate(actual.On.X)
	case *query.Union:
		return hasCollate(actual.X)
	}
	return false
}

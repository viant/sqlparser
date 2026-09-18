package sqlparser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/viant/sqlparser/column"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/index"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/table"
	"github.com/viant/sqlparser/update"
)

// Stringify retains the established SQL rendering contract.
func Stringify(n node.Node) string { return (Stringifier{}).String(n) }

// Stringifier configures AST rendering without changing existing consumers.
type Stringifier struct{ PreserveWindow bool }

// String renders a node, optionally retaining authored LIMIT/OFFSET clauses.
func (s Stringifier) String(n node.Node) string {
	builder := new(bytes.Buffer)
	s.append(n, builder)
	return builder.String()
}

func (s Stringifier) append(n node.Node, builder *bytes.Buffer) {
	if n == nil {
		panic("node was nill")
	}
	switch actual := n.(type) {
	case string:
		builder.WriteString(actual)
	case *query.Select:
		if len(actual.WithSelects) > 0 {
			if actual.WithRecursive {
				builder.WriteString("WITH RECURSIVE ")
			} else {
				builder.WriteString("WITH ")
			}
			for i, withSel := range actual.WithSelects {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(withSel.Alias)
				builder.WriteString(" AS ")
				switch {
				case withSel.Raw != "":
					builder.WriteString(withSel.Raw)
				case withSel.X != nil:
					builder.WriteByte('(')
					s.append(withSel.X, builder)
					builder.WriteByte(')')
				default:
					builder.WriteString("()")
				}
			}
			builder.WriteByte(' ')
		}
		builder.WriteString("SELECT ")
		if kind := strings.TrimSpace(actual.Kind); kind != "" {
			builder.WriteString(kind)
			builder.WriteByte(' ')
		}
		s.append(actual.List, builder)
		if actual.From.X != nil {
			builder.WriteString(" FROM ")
			s.append(&actual.From, builder)
		}

		if len(actual.Joins) > 0 {
			for _, join := range actual.Joins {
				s.append(join, builder)
			}
		}
		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			s.append(actual.Qualify.X, builder)
		}
		if len(actual.GroupBy) > 0 {
			builder.WriteString(" GROUP BY ")
			for i, item := range actual.GroupBy {
				if i != 0 {
					builder.WriteString(", ")
				}
				s.append(item, builder)
			}
		}
		if actual.Having != nil {
			builder.WriteString(" HAVING ")
			s.append(actual.Having, builder)
		}

		if len(actual.OrderBy) > 0 {
			builder.WriteString(" ORDER BY ")
			for i, item := range actual.OrderBy {
				if i > 0 {
					builder.WriteString(", ")
				}
				s.append(item, builder)
			}
		}
		if s.PreserveWindow && actual.Limit != nil {
			builder.WriteString(" LIMIT ")
			builder.WriteString(Stringify(actual.Limit))
		}
		if s.PreserveWindow && actual.Offset != nil {
			builder.WriteString(" OFFSET ")
			builder.WriteString(Stringify(actual.Offset))
		}
		if union := actual.Union; union != nil {
			builder.WriteString(" " + union.Raw + " ")
			s.append(union.X, builder)
		}

	case *query.Join:
		builder.WriteByte(' ')
		builder.WriteString(actual.Raw)
		builder.WriteByte(' ')

		s.append(actual.With, builder)
		if actual.Alias != "" {
			builder.WriteByte(' ')
			builder.WriteString(actual.Alias)
		}
		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}
		if actual.On != nil {
			builder.WriteString(" ON ")
			s.append(actual.On, builder)
		}
	case *expr.Qualify:
		s.append(actual.X, builder)
	case *expr.Literal:
		builder.WriteString(actual.Value)
	case query.List:
		listSize := len(actual)
		if listSize == 0 {
			return
		}
		s.append(actual[0], builder)
		for i := 1; i < listSize; i++ {
			builder.WriteString(", ")
			s.append(actual[i], builder)
		}

	case *expr.Star:
		s.append(actual.X, builder)
		if len(actual.Except) > 0 {
			builder.WriteString(" EXCEPT ")
			if len(actual.Except) > 1 {
				builder.WriteString("(")
			}
			for i, item := range actual.Except {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(item)
			}
			if len(actual.Except) > 1 {
				builder.WriteString(")")
			}
		}
		if actual.Comments != "" {
			builder.WriteString(" ")
			builder.WriteString(actual.Comments)

		}
	case *expr.Raw:
		builder.WriteString(" ")
		builder.WriteString(actual.Raw)
		builder.WriteString(" ")
		builder.WriteString(actual.Unparsed)
	case *expr.Collate:
		s.append(actual.X, builder)
		builder.WriteString(" COLLATE ")
		builder.WriteString(actual.Collation)
	case *query.From:
		if actual.X == nil {
			return
		}
		s.append(actual.X, builder)
		if actual.Alias != "" {
			builder.WriteString(" " + actual.Alias)
		}

		if actual.Unparsed != "" {
			builder.WriteString(" ")
			builder.WriteString(actual.Unparsed)
		}

		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}

	case *expr.Placeholder:
		builder.WriteString(actual.Name)
	case *expr.Unary:
		builder.WriteString(" " + actual.Op + " ")
		s.append(actual.X, builder)
	case *expr.Parenthesis:
		builder.WriteString(actual.Raw)
	case *expr.Switch:
		if actual.Raw != "" {
			builder.WriteString(actual.Raw)
			break
		}
		builder.WriteString("CASE")
		if actual.Ident.Name != "" {
			builder.WriteString(" ")
			builder.WriteString(actual.Ident.Name)
		}
		for _, candidate := range actual.Cases {
			if candidate == nil {
				continue
			}
			if candidate.Y == nil {
				builder.WriteByte(' ')
				s.append(candidate.X.X, builder)
				continue
			}
			if candidate.X.X == nil {
				builder.WriteString(" ELSE ")
				s.append(candidate.Y, builder)
				continue
			}
			builder.WriteString(" WHEN ")
			s.append(candidate.X.X, builder)
			builder.WriteString(" THEN ")
			s.append(candidate.Y, builder)
		}
		builder.WriteString(" END")
	case []node.Node:
		for i, item := range actual {
			if i > 0 {
				builder.WriteString(", ")
			}
			s.append(item, builder)
		}
	case *query.Item:
		s.append(actual.Expr, builder)
		if actual.Alias != "" {
			builder.WriteString(" AS " + actual.Alias)
		}
		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}
		if actual.Direction != "" {
			builder.WriteString(" " + actual.Direction)
		}
	case *expr.Binary:
		s.append(actual.X, builder)
		builder.WriteString(" ")
		if actual.Op != "" {
			builder.WriteString(actual.Op + " ")
		}
		if actual.Y != nil {
			s.append(actual.Y, builder)
		}
	case expr.Raw:
		builder.WriteString(actual.Raw)
		builder.WriteString(actual.Unparsed)
	case *expr.Ident:
		builder.WriteString(actual.Name)
	case *expr.Call:
		s.append(actual.X, builder)
		builder.WriteString(actual.Raw)
	case *expr.Subscript:
		s.append(actual.X, builder)
		builder.WriteByte('[')
		s.append(actual.Index, builder)
		builder.WriteByte(']')
	case *expr.FieldAccess:
		s.append(actual.X, builder)
		builder.WriteByte('.')
		builder.WriteString(actual.Name)
	case *expr.NullTreatment:
		s.append(actual.X, builder)
		builder.WriteString(" " + actual.Mode + " NULLS")
	case *expr.Range:
		s.append(actual.Min, builder)
		builder.WriteString(" AND ")
		s.append(actual.Max, builder)
	case *expr.Selector:
		name := actual.Name
		if actual.Expression == "" {
			builder.WriteString(name)
		} else if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") && len(name) > 1 {
			builder.WriteString(name[:len(name)-1])
			builder.WriteByte('[')
			builder.WriteString(actual.Expression)
			builder.WriteString("]`")
		} else {
			builder.WriteString(name)
			builder.WriteByte('[')
			builder.WriteString(actual.Expression)
			builder.WriteByte(']')
		}
		if actual.X != nil {
			builder.WriteByte('.')
			s.append(actual.X, builder)
		}
	case *update.Item:
		s.append(actual.Column, builder)
		builder.WriteString(" = ")
		s.append(actual.Expr, builder)
	case *update.Statement:
		builder.WriteString("UPDATE ")
		s.append(actual.Target.X, builder)
		builder.WriteString(" SET ")
		for i := range actual.Set {
			if i > 0 {
				builder.WriteString(", ")
			}
			s.append(actual.Set[i], builder)
		}
		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			s.append(actual.Qualify, builder)
		}
	case *insert.Statement:
		builder.WriteString("INSERT INTO ")
		s.append(actual.Target.X, builder)
		builder.WriteString(" (")
		builder.WriteString(strings.Join(actual.Columns, ", "))
		builder.WriteString(") VALUES(")
		valuesLen := len(actual.Values)
		columnLen := len(actual.Columns)
		for i := 0; i < valuesLen; i += columnLen {
			if i > 0 {
				builder.WriteString("), (")
			}
			for j := 0; j < columnLen; j++ {
				if j > 0 {
					builder.WriteString(", ")
				}
				s.append(actual.Values[i+j].Expr, builder)
			}
		}
		builder.WriteString(")")
		if actual.Alias != "" {
			builder.WriteString(" AS " + actual.Alias)
			if len(actual.OnDuplicateKeyUpdate) > 0 {
				builder.WriteString(" ON DUPLICATE KEY UPDATE ")
				for i, item := range actual.OnDuplicateKeyUpdate {
					if i > 0 {
						builder.WriteString(", ")
					}
					s.append(item, builder)
				}
			}
		}
	case *del.Statement:
		builder.WriteString("DELETE")
		for i, item := range actual.Items {
			if i != 0 {
				builder.WriteString(", ")
			}

			s.append(item, builder)
		}

		s.append(actual.Target, builder)
		for _, join := range actual.Joins {
			s.append(join, builder)
		}

		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			s.append(actual.Qualify, builder)
		}
	case del.Target:
		builder.WriteString(" FROM ")
		s.append(actual.X, builder)
		if actual.Alias != "" {
			builder.WriteString(" " + actual.Alias)
		}

		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}
	case *del.Item:
		builder.WriteString(" ")
		builder.WriteString(actual.Raw)
		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}
	case *table.Create:
		builder.WriteString("CREATE TABLE ")
		if actual.IfDoesExists {
			builder.WriteString("IF NOT EXISTS ")
		}
		builder.WriteString(actual.Name)
		builder.WriteString("(\n")
		for i, col := range actual.Columns {
			if i > 0 {
				builder.WriteString(",\n")
			}
			s.append(col, builder)
		}
		builder.WriteString(")")

	case *index.Create:
		builder.WriteString("CREATE ")
		if actual.Type != "" {
			builder.WriteString(actual.Type)
			builder.WriteString(" ")
		}

		builder.WriteString("INDEX ")

		if actual.IfDoesExists {
			builder.WriteString("IF NOT EXISTS ")
		}
		builder.WriteString(actual.Name)
		builder.WriteString(" ON ")

		if actual.Schema != "" {
			builder.WriteString(actual.Schema)
			builder.WriteString(".")
			builder.WriteString(actual.Table)
		} else {
			builder.WriteString(actual.Table)
		}

		builder.WriteString("(")
		for i, col := range actual.Columns {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(col.Name)
		}
		builder.WriteString(");")

	case *column.Spec:
		builder.WriteString(actual.Name)
		builder.WriteString(" ")
		builder.WriteString(actual.Type)
		if actual.Key != "" {
			builder.WriteString(" ")
			builder.WriteString(actual.Key)
		}
		if !actual.IsNullable {
			builder.WriteString(" NOT NULL")
		}
		if actual.Default != nil {
			builder.WriteString(" ")
			builder.WriteString("DEFAULT ")
			builder.WriteString(*actual.Default)
		}

	default:
		panic(fmt.Sprintf("%T unsupported", n))
	}

}

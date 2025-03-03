package sqlparser

import (
	"bytes"
	"fmt"
	"github.com/viant/sqlparser/column"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/index"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/table"
	"github.com/viant/sqlparser/update"
	"strings"
)

// Stringify stringifies node
func Stringify(n node.Node) string {
	builder := new(bytes.Buffer)
	stringify(n, builder)
	return builder.String()
}

func stringify(n node.Node, builder *bytes.Buffer) {
	if n == nil {
		panic("node was nill")
	}
	switch actual := n.(type) {
	case string:
		builder.WriteString(actual)
	case *query.Select:
		builder.WriteString("SELECT ")
		stringify(actual.List, builder)
		builder.WriteString(" FROM ")
		stringify(&actual.From, builder)

		if len(actual.Joins) > 0 {
			for _, join := range actual.Joins {
				stringify(join, builder)
			}
		}
		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			stringify(actual.Qualify.X, builder)
		}
		if len(actual.GroupBy) > 0 {
			builder.WriteString(" GROUP BY ")
			for i, item := range actual.GroupBy {
				if i != 0 {
					builder.WriteString(", ")
				}
				stringify(item, builder)
			}
		}
		if actual.Having != nil {
			builder.WriteString(" HAVING ")
			stringify(actual.Having, builder)
		}

		if len(actual.OrderBy) > 0 {
			builder.WriteString(" ORDER BY ")
			for i, item := range actual.OrderBy {
				if i > 0 {
					builder.WriteString(", ")
				}
				stringify(item, builder)
			}
		}
		if union := actual.Union; union != nil {
			builder.WriteString(" " + union.Raw + " ")
			stringify(union.X, builder)
		}

	case *query.Join:
		builder.WriteByte(' ')
		builder.WriteString(actual.Raw)
		builder.WriteByte(' ')

		stringify(actual.With, builder)
		if actual.Alias != "" {
			builder.WriteByte(' ')
			builder.WriteString(actual.Alias)
		}
		if actual.Comments != "" {
			builder.WriteString(" " + actual.Comments)
		}
		if actual.On != nil {
			builder.WriteString(" ON ")
			stringify(actual.On, builder)
		}
	case *expr.Qualify:
		stringify(actual.X, builder)
	case *expr.Literal:
		builder.WriteString(actual.Value)
	case query.List:
		listSize := len(actual)
		if listSize == 0 {
			return
		}
		stringify(actual[0], builder)
		for i := 1; i < listSize; i++ {
			builder.WriteString(", ")
			stringify(actual[i], builder)
		}

	case *expr.Star:
		stringify(actual.X, builder)
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
	case *query.From:
		if actual.X == nil {
			return
		}
		stringify(actual.X, builder)
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
		stringify(actual.X, builder)
	case *expr.Parenthesis:
		builder.WriteString(actual.Raw)
	case *query.Item:
		stringify(actual.Expr, builder)
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
		stringify(actual.X, builder)
		builder.WriteString(" ")
		if actual.Op != "" {
			builder.WriteString(actual.Op + " ")
		}
		if actual.Y != nil {
			stringify(actual.Y, builder)
		}
	case expr.Raw:
		builder.WriteString(actual.Raw)
		builder.WriteString(actual.Unparsed)
	case *expr.Ident:
		builder.WriteString(actual.Name)
	case *expr.Call:
		stringify(actual.X, builder)
		builder.WriteString(actual.Raw)
	case *expr.Range:
		stringify(actual.Min, builder)
		builder.WriteString(" AND ")
		stringify(actual.Max, builder)
	case *expr.Selector:
		builder.WriteString(actual.Name)
		if actual.Expression != "" {
			builder.WriteString("[")
			builder.WriteString(actual.Expression)
			builder.WriteString("]")
		}
		if actual.X != nil {
			builder.WriteByte('.')
		}
		stringify(actual.X, builder)
	case *update.Item:
		stringify(actual.Column, builder)
		builder.WriteString(" = ")
		stringify(actual.Expr, builder)
	case *update.Statement:
		builder.WriteString("UPDATE ")
		stringify(actual.Target.X, builder)
		builder.WriteString(" SET ")
		for i := range actual.Set {
			if i > 0 {
				builder.WriteString(", ")
			}
			stringify(actual.Set[i], builder)
		}
		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			stringify(actual.Qualify, builder)
		}
	case *insert.Statement:
		builder.WriteString("INSERT INTO ")
		stringify(actual.Target.X, builder)
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
				stringify(actual.Values[i+j].Expr, builder)
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
					stringify(item, builder)
				}
			}
		}
	case *del.Statement:
		builder.WriteString("DELETE")
		for i, item := range actual.Items {
			if i != 0 {
				builder.WriteString(", ")
			}

			stringify(item, builder)
		}

		stringify(actual.Target, builder)
		for _, join := range actual.Joins {
			stringify(join, builder)
		}

		if actual.Qualify != nil {
			builder.WriteString(" WHERE ")
			stringify(actual.Qualify, builder)
		}
	case del.Target:
		builder.WriteString(" FROM ")
		stringify(actual.X, builder)
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
			stringify(col, builder)
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

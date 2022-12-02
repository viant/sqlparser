package sqlparser

import (
	"bytes"
	"fmt"
	del "github.com/viant/sqlx/metadata/ast/delete"
	"github.com/viant/sqlx/metadata/ast/expr"
	"github.com/viant/sqlx/metadata/ast/insert"
	"github.com/viant/sqlx/metadata/ast/node"
	"github.com/viant/sqlx/metadata/ast/query"
	"github.com/viant/sqlx/metadata/ast/update"
	"strings"
)

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

		if len(actual.OrderBy) > 0 {
			builder.WriteString(" ORDER BY ")
			for _, item := range actual.OrderBy {
				stringify(item, builder)
			}
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
		builder.WriteString(" ON ")
		stringify(actual.On, builder)
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
			for i, item := range actual.Except {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(item)
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
	case *query.From:
		if actual.X == nil {
			return
		}
		stringify(actual.X, builder)
		if actual.Alias != "" {
			builder.WriteString(" " + actual.Alias)
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
		builder.WriteString(" " + actual.Op + " ")
		if actual.Y != nil {
			stringify(actual.Y, builder)
		}
	case expr.Raw:
		builder.WriteString(actual.Raw)
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
		for i, value := range actual.Values {
			if i > 0 {
				builder.WriteString(", ")
			}
			stringify(value.Expr, builder)
		}
		builder.WriteString(")")
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
	default:
		panic(fmt.Sprintf("%T unsupported", n))
	}

}

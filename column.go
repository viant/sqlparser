package sqlparser

import (
	"github.com/viant/sqlparser/column"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
	"strings"
)

//Column represent column
type Column column.Spec

func NewColumn(item *query.Item) *Column {
	switch actual := item.Expr.(type) {
	case *expr.Call:
		call := Stringify(actual)
		fn, args := extractArguments(call)
		lcArgs := strings.ToLower(args)
		if item.DataType == "" {
			item.DataType = "string"
		}
		name := ""
		if strings.EqualFold(fn, "cast") {
			if index := strings.Index(lcArgs, " as "); index != -1 {
				name = strings.TrimSpace(args[:index])
				targetType := strings.TrimSpace(args[index+4:])
				item.DataType = adjustDataType(targetType)
			}
		}
		if strings.EqualFold(fn, "tag") {
			params := strings.SplitN(args, ",", 2)
			name = params[0]
			item.Tag = params[1]
		}
		return &Column{Name: name, Alias: item.Alias, Type: item.DataType, Tag: item.Tag, Expression: call}
	case *expr.Ident:
		return &Column{Name: actual.Name, Alias: item.Alias, Type: item.DataType, Tag: item.Tag}
	case *expr.Selector:
		return &Column{Name: Stringify(actual.X), Namespace: actual.Name, Type: item.DataType, Alias: item.Alias, Tag: item.Tag}
	case *expr.Star:
		switch star := actual.X.(type) {
		case *expr.Ident:
			return &Column{Namespace: star.Name, Except: actual.Except, Tag: item.Tag, Expression: Stringify(star)}
		case *expr.Selector:
			return &Column{Namespace: star.Name, Except: actual.Except, Comments: actual.Comments, Tag: item.Tag, Expression: Stringify(star)}
		}
	case *expr.Literal:
		return &Column{Name: "", Alias: item.Alias, Type: actual.Kind, Tag: item.Tag, Expression: actual.Value}
	case *expr.Binary:
		expression := Stringify(actual)
		if item.DataType == "" || (strings.Contains(expression, "+") || strings.Contains(expression, "-") || strings.Contains(expression, "/") || strings.Contains(expression, "*")) {
			item.DataType = "float64"
		}
		return &Column{Alias: item.Alias, Type: item.DataType, Tag: item.Tag, Expression: expression}
	case *expr.Parenthesis:
		return &Column{Name: Stringify(actual), Alias: item.Alias, Type: item.DataType, Tag: item.Tag, Expression: actual.Raw}
	}
	return &Column{Name: item.Raw, Expression: Stringify(item.Expr), Alias: item.Alias, Comments: item.Comments}
}

func extractArguments(expr string) (string, string) {
	fn := ""
	if index := strings.Index(expr, "("); index != -1 {
		fn = expr[:index]
		expr = expr[index+1:]
	}
	if index := strings.LastIndex(expr, ")"); index != -1 {
		expr = expr[:index]
	}
	return fn, expr
}

func adjustDataType(targetType string) string {
	switch strings.ToLower(targetType) {
	case "signed":
		return "int"
	}
	return "string"
}

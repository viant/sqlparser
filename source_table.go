package sqlparser

import (
	"fmt"
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

// SourceTable identifies one complete table reference and whether parentheses
// enclose it. Subqueries and other source expressions return an empty name.
// It uses the SQL table/parenthesis token owners, never keyword text heuristics,
// and does not mutate the supplied AST (including partially populated Raw.X).
func SourceTable(value node.Node) (table string, parenthesized bool, err error) {
	var raw string
	switch actual := value.(type) {
	case nil:
		return "", false, nil
	case *expr.Ident:
		if actual == nil {
			return "", false, fmt.Errorf("nil table identifier")
		}
		raw = actual.Name
	case *expr.Selector:
		if actual == nil {
			return "", false, fmt.Errorf("nil table selector")
		}
		raw = Stringify(actual)
	case *expr.Raw:
		if actual == nil {
			return "", false, fmt.Errorf("nil table source")
		}
		raw = actual.Raw
	case *expr.Parenthesis:
		if actual == nil {
			return "", false, fmt.Errorf("nil parenthesized source")
		}
		raw = actual.Raw
	default:
		return "", false, nil
	}
	for {
		cursor := parsly.NewCursor("table source", []byte(raw), 0)
		// Placeholder recognition must precede the permissive table matcher:
		// that matcher also accepts template selectors such as $View.X.SQL.
		match := cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, placeholderMatcher, tableMatcher, doubleQuotedStringLiteralMatcher)
		if match.Code != parenthesesCode && match.Code != tableTokenCode && match.Code != doubleQuotedStringLiteral {
			return "", false, nil
		}
		text := match.Text(cursor)
		if strings.TrimSpace(string(cursor.Input[cursor.Pos:])) != "" {
			return "", false, nil
		}
		if match.Code == tableTokenCode || match.Code == doubleQuotedStringLiteral {
			return text, parenthesized, nil
		}
		parenthesized = true
		raw = text[1 : len(text)-1]
	}
}

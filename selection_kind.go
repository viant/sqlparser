package sqlparser

import (
	"strings"

	"github.com/viant/parsly"
)

// Keep the complete SELECT modifier in Kind so native AST rendering preserves
// both DISTINCT/ALL and BigQuery's AS STRUCT projection mode.
func parseSelectKind(cursor *parsly.Cursor) (kind string, asStruct bool, err error) {
	skipExpressionSpace(cursor)
	if match := cursor.MatchOne(selectionKindMatcher); match.Code == selectionKindCode {
		kind = match.Text(cursor)
	}
	skipExpressionSpace(cursor)
	pos := cursor.Pos
	match := cursor.MatchOne(selectAsKeywordMatcher)
	if match.Code != asKeyword {
		cursor.Pos = pos
		return kind, false, nil
	}
	as := match.Text(cursor)
	skipExpressionSpace(cursor)
	match = cursor.MatchOne(selectStructKeywordMatcher)
	if match.Code != selectionKindCode || strings.EqualFold(kind, "STRUCT") {
		return "", false, cursor.NewError(exprMatcher)
	}
	if kind != "" {
		kind += " "
	}
	kind += as + " " + match.Text(cursor)
	skipExpressionSpace(cursor)
	return kind, true, nil
}

// selectionModifier shares the native selector boundary, so ordinary names
// such as all_records and distinct_column are not consumed as modifiers.
type selectionModifier struct {
	keywords          parsly.Matcher
	selector          parsly.Matcher
	excludeStructCall bool
}

func (m selectionModifier) Match(cursor *parsly.Cursor) int {
	matched := m.keywords.Match(cursor)
	if matched != 0 && m.selector.Match(cursor) > matched && !startsSQLComment(cursor.Input, cursor.Pos+matched) {
		return 0
	}
	// STRUCT(...) is an expression, including at the start of a projection
	// or argument list. The explicit AS STRUCT matcher still accepts it as
	// a query modifier followed by a parenthesized projection.
	if m.excludeStructCall && matched > 0 && strings.EqualFold(string(cursor.Input[cursor.Pos:cursor.Pos+matched]), "STRUCT") {
		lookahead := *cursor
		lookahead.Pos += matched
		skipExpressionSpace(&lookahead)
		if lookahead.Pos < len(lookahead.Input) && lookahead.Input[lookahead.Pos] == '(' {
			return 0
		}
	}
	return matched
}

// Comments separate SQL tokens even when no whitespace surrounds them.
func startsSQLComment(input []byte, pos int) bool {
	return pos+1 < len(input) && (input[pos] == '/' && input[pos+1] == '*' || input[pos] == '-' && input[pos+1] == '-')
}

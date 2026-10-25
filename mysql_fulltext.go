package sqlparser

import (
	"fmt"
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/source"
)

// MATCH is a native call; its AGAINST operand contains a search expression and
// optional MySQL mode keywords. Preserve both expression trees and authored mode
// text so projection/identifier analysis does not need an opaque SQL fallback.
func parseMatchAgainst(cursor *parsly.Cursor, operand node.Node) (node.Node, error) {
	call, ok := operand.(*expr.Call)
	if !ok || !strings.EqualFold(Stringify(call.X), "MATCH") {
		return operand, nil
	}
	tail := *cursor
	skipExpressionSpace(&tail)
	keyword := tail.MatchOne(selectorMatcher)
	if keyword.Code != selectorTokenCode || !strings.EqualFold(keyword.Text(&tail), "AGAINST") {
		return operand, nil
	}
	if len(call.Args) == 0 {
		return nil, fmt.Errorf("MATCH requires at least one column")
	}
	match := matchCallParentheses(&tail)
	if match.Code != parenthesesCode {
		return nil, fmt.Errorf("AGAINST requires a parenthesized search expression")
	}
	raw := match.Text(&tail)
	inner := raw[1 : len(raw)-1]
	modeIndex := -1
	for _, mode := range []string{"IN NATURAL LANGUAGE MODE", "IN BOOLEAN MODE", "WITH QUERY EXPANSION"} {
		if index := source.FindTopLevelKeyword(inner, mode, 0); index >= 0 && (modeIndex < 0 || index < modeIndex) {
			modeIndex = index
		}
	}
	searchText := inner
	modeText := ""
	if modeIndex >= 0 {
		searchText = inner[:modeIndex]
		modeCursor := parsly.NewCursor(cursor.Path, []byte(inner[modeIndex:]), 0)
		var words []string
		for {
			skipExpressionSpace(modeCursor)
			if modeCursor.Pos == len(modeCursor.Input) {
				break
			}
			word := modeCursor.MatchOne(selectorMatcher)
			if word.Code != selectorTokenCode {
				return nil, fmt.Errorf("invalid AGAINST search mode")
			}
			words = append(words, strings.ToUpper(word.Text(modeCursor)))
		}
		modeText = strings.Join(words, " ")
		switch modeText {
		case "IN NATURAL LANGUAGE MODE", "IN NATURAL LANGUAGE MODE WITH QUERY EXPANSION", "IN BOOLEAN MODE", "WITH QUERY EXPANSION":
		default:
			return nil, fmt.Errorf("invalid AGAINST search mode")
		}
	}
	searchCursor := parsly.NewCursor(cursor.Path, []byte(searchText), 0)
	search, err := expectExpression(searchCursor)
	if err != nil {
		return nil, err
	}
	skipExpressionSpace(searchCursor)
	if search == nil || searchCursor.Pos != len(searchCursor.Input) {
		return nil, fmt.Errorf("invalid AGAINST search expression")
	}
	if modeText != "" {
		// Keep the suffix in the semantic tree as well as Raw. Transforms that
		// rebuild parentheses must not silently drop the selected search mode.
		search = &expr.Binary{X: search, Op: modeText}
	}
	cursor.Pos = tail.Pos
	return applyCollate(cursor, &expr.Binary{X: operand, Op: "AGAINST", Y: &expr.Parenthesis{Raw: raw, X: search}})
}

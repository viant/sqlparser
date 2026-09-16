package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/query"
)

func parseExistsQuery(cursor *parsly.Cursor) (*query.Select, error) {
	// Parentheses may enclose the query repeatedly, but each enclosure must
	// contain the entire argument. The enclosing call retains its raw syntax.
	for {
		skipExpressionSpace(cursor)
		match := cursor.MatchOne(parenthesesMatcher)
		if match.Code != parenthesesCode {
			break
		}
		raw := match.Text(cursor)
		start := cursor.Pos - len(raw)
		skipExpressionSpace(cursor)
		if cursor.Pos != len(cursor.Input) {
			return nil, cursor.NewError(exprMatcher)
		}
		inner := parsly.NewCursor(cursor.Path, []byte(raw[1:len(raw)-1]), start)
		inner.OnError = cursor.OnError
		cursor = inner
	}
	start := cursor.Pos
	match := cursor.MatchAny(selectKeywordMatcher, withKeywordMatcher)
	if match.Code != selectKeyword && match.Code != withKeyword {
		return nil, cursor.NewError(selectKeywordMatcher)
	}
	cursor.Pos = start
	result := &query.Select{}
	if err := parseQuery(cursor, result); err != nil {
		return nil, err
	}
	skipExpressionSpace(cursor)
	if cursor.Pos != len(cursor.Input) || !completeQueryProjections(result) {
		return nil, cursor.NewError(exprMatcher)
	}
	return result, nil
}

// The general query parser permits incomplete projections for legacy callers.
// EXISTS requires a complete projection in its main query, CTEs and UNION arms.
func completeQueryProjections(q *query.Select) bool {
	if q == nil || len(q.List) == 0 {
		return false
	}
	for _, item := range q.List {
		if item == nil || !completeExpression(item.Expr) {
			return false
		}
	}
	for _, with := range q.WithSelects {
		if with == nil || !completeQueryProjections(with.X) {
			return false
		}
	}
	return q.Union == nil || completeQueryProjections(q.Union.X)
}

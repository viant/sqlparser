package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

// STRUCT fields are expressions with optional AS names. Named fields reuse
// query.Item so serialization, traversal and transformations retain the name.
func parseStructArguments(cursor *parsly.Cursor) ([]node.Node, error) {
	var args []node.Node
	skipExpressionSpace(cursor)
	if cursor.Pos == len(cursor.Input) {
		return args, nil
	}
	for {
		field, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		skipExpressionSpace(cursor)
		if cursor.MatchOne(selectAsKeywordMatcher).Code == asKeyword {
			skipExpressionSpace(cursor)
			alias := cursor.MatchOne(aliasIdentifierMatcher)
			if alias.Code != identifierCode || !(aliasIdentifier{}).boundary(cursor) {
				return nil, cursor.NewError(aliasIdentifierMatcher)
			}
			field = &query.Item{Expr: field, Alias: alias.Text(cursor)}
		}
		args = append(args, field)
		skipExpressionSpace(cursor)
		if cursor.Pos == len(cursor.Input) {
			return args, nil
		}
		if cursor.MatchOne(nextMatcher).Code != nextCode {
			return nil, cursor.NewError(nextMatcher)
		}
	}
}

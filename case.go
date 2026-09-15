package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
)

// parseCase consumes the same expression cursor recursively. Raw is retained
// for unchanged rendering; every executable expression also has an AST node.
func parseCase(cursor *parsly.Cursor, start int) (*expr.Switch, error) {
	result := &expr.Switch{}
	skipExpressionSpace(cursor)
	match := cursor.MatchOne(whenKeywordMatcher)
	if match.Code != whenKeyword {
		selector, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		result.Cases = append(result.Cases, &expr.Case{X: expr.Qualify{X: selector}})
		skipExpressionSpace(cursor)
		match = cursor.MatchOne(whenKeywordMatcher)
	}
	if match.Code != whenKeyword {
		return nil, cursor.NewError(whenKeywordMatcher)
	}
	for {
		condition, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		skipExpressionSpace(cursor)
		if cursor.MatchOne(thenKeywordMatcher).Code != thenKeyword {
			return nil, cursor.NewError(thenKeywordMatcher)
		}
		value, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		result.Cases = append(result.Cases, &expr.Case{X: expr.Qualify{X: condition}, Y: value})
		skipExpressionSpace(cursor)
		match = cursor.MatchAny(whenKeywordMatcher, elseKeywordMatcher, endKeywordMatcher)
		if match.Code == whenKeyword {
			continue
		}
		if match.Code == elseKeyword {
			value, err = expectExpression(cursor)
			if err != nil {
				return nil, err
			}
			result.Cases = append(result.Cases, &expr.Case{Y: value})
			skipExpressionSpace(cursor)
			match = cursor.MatchOne(endKeywordMatcher)
		}
		if match.Code != endKeyword {
			return nil, cursor.NewError(endKeywordMatcher)
		}
		result.Raw = string(cursor.Input[start:cursor.Pos])
		return result, nil
	}
}

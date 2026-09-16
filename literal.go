package sqlparser

import (
	"bytes"
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/source"
)

// ParseLiteral parses literal
func ParseLiteral(cursor *parsly.Cursor) (*expr.Literal, error) {
	return parseLiteral(cursor, true)
}

// TryParseLiteral tries to parse literal
func TryParseLiteral(cursor *parsly.Cursor) (*expr.Literal, error) {
	return parseLiteral(cursor, false)
}

var literalTokens = []*parsly.Token{
	asKeywordMatcher,
	nextMatcher,
	nullKeywordMatcher,
	boolLiteralMatcher,
	rawSingleQuotedStringLiteralMatcher,
	doubleQuotedStringLiteralMatcher,
	singleQuotedStringLiteralMatcher,
	intLiteralMatcher,
	numericLiteralMatcher,
}

func parseLiteral(cursor *parsly.Cursor, shallRaiseInvalidToken bool) (*expr.Literal, error) {
	match := cursor.MatchAfterOptional(whitespaceMatcher, literalTokens...)
	switch match.Code {
	case asKeyword, nextCode:
		cursor.Pos -= match.Size
		return nil, nil
	case nullKeyword:
		return expr.NewNullLiteral(match.Text(cursor)), nil
	case singleQuotedStringLiteral, rawSingleQuotedStringLiteral, doubleQuotedStringLiteral:
		return expr.NewStringLiteral(match.Text(cursor)), nil
	case boolLiteral:
		return expr.NewBoolLiteral(match.Text(cursor)), nil
	case intLiteral:
		return expr.NewIntLiteral(match.Text(cursor)), nil
	case numericLiteral:
		return expr.NewNumericLiteral(match.Text(cursor)), nil
	case parsly.EOF:
		return nil, nil
	case parsly.Invalid:
		// Consume dollar-quoted text as a whole operand, just as the source
		// scanner does, so projection boundary checks do not stop inside it.
		if cursor.Pos < len(cursor.Input) && cursor.Input[cursor.Pos] == '$' {
			text := cursor.Input[cursor.Pos:]
			if size := source.DollarQuoteDelimiterSize(text); size > 0 {
				close := bytes.Index(text[size:], text[:size])
				if close < 0 {
					return nil, fmt.Errorf("unclosed SQL quoted text at byte %d", cursor.Pos)
				}
				end := size + close + size
				cursor.Pos += end
				return expr.NewStringLiteral(string(text[:end])), nil
			}
		}
		if shallRaiseInvalidToken {
			return nil, cursor.NewError(literalTokens...)
		}
	}
	return nil, nil
}

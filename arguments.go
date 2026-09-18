package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"strings"
)

func skipExpressionSpace(cursor *parsly.Cursor) {
	for {
		match := cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher, inlineCommentMatcher)
		if match.Code != commentBlock {
			return
		}
	}
}

func expectExpression(cursor *parsly.Cursor) (node.Node, error) {
	skipExpressionSpace(cursor)
	binary := &expr.Binary{}
	if err := parseBinaryExpr(cursor, binary); err != nil {
		return nil, err
	}
	if binary.X == nil || !completeExpression(binary) {
		return nil, cursor.NewError(exprMatcher)
	}
	if binary.Op == "" && binary.Y == nil {
		return binary.X, nil
	}
	return binary, nil
}

func completeExpression(n node.Node) bool {
	switch actual := n.(type) {
	case nil:
		return false
	case *expr.Binary:
		return completeExpression(actual.X) && (actual.Op == "" && actual.Y == nil || completeExpression(actual.Y))
	case *expr.Unary:
		return completeExpression(actual.X)
	case *expr.Range:
		return completeExpression(actual.Min) && completeExpression(actual.Max)
	}
	return true
}

// matchArgumentName recognizes => only at a function argument boundary. Argument
// names are syntax, not column references; their values remain ordinary expressions.
func matchArgumentName(cursor *parsly.Cursor) string {
	skipExpressionSpace(cursor)
	pos := cursor.Pos
	match := cursor.MatchOne(aliasIdentifierMatcher)
	if match.Code == identifierCode {
		name := match.Text(cursor)
		skipExpressionSpace(cursor)
		if strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "=>") {
			cursor.Pos += 2
			return name
		}
	}
	cursor.Pos = pos
	return ""
}

func parseArgumentList(cursor *parsly.Cursor, list *query.List, ordered, allowNamed bool) error {
	skipExpressionSpace(cursor)
	if cursor.Pos == len(cursor.Input) && !ordered {
		return nil
	}
	var names map[string]bool
	for {
		if allowNamed {
			if name := matchArgumentName(cursor); name != "" {
				parts, err := TableIdentifierParts(name)
				if err != nil {
					return err
				}
				key := strings.ToLower(parts[0])
				if names[key] {
					return fmt.Errorf("duplicate named argument: %s", name)
				}
				if names == nil {
					names = make(map[string]bool)
				}
				names[key] = true
				value, err := expectExpression(cursor)
				if err != nil {
					return err
				}
				list.Append(query.NewItem(&expr.Binary{X: &expr.Raw{Raw: name}, Op: "=>", Y: value}))
				skipExpressionSpace(cursor)
				if cursor.Pos == len(cursor.Input) {
					return nil
				}
				if cursor.MatchOne(nextMatcher).Code != nextCode {
					return cursor.NewError(nextMatcher)
				}
				continue
			}
			if len(names) != 0 {
				return fmt.Errorf("expected named argument at byte %d", cursor.Pos)
			}
		}
		modifier := ""
		if len(*list) == 0 && !ordered {
			if match := cursor.MatchOne(selectionKindMatcher); match.Code == selectionKindCode {
				modifier = match.Text(cursor)
			}
		}
		operand, err := expectExpression(cursor)
		if err != nil {
			return err
		}
		if modifier != "" {
			operand = &expr.Unary{Op: modifier, X: operand}
		}
		item := query.NewItem(operand)
		list.Append(item)
		hasSeparator := false
		skipExpressionSpace(cursor)
		if !ordered {
			if match := cursor.MatchAfterOptional(whitespaceMatcher, ignoreKeywordMatcher, respectKeywordMatcher); match.Code == nullTreatmentKeyword {
				mode := strings.ToUpper(match.Text(cursor))
				skipExpressionSpace(cursor)
				if cursor.MatchOne(nullsKeywordMatcher).Code != nullsKeyword {
					return cursor.NewError(nullsKeywordMatcher)
				}
				item.Expr = &expr.NullTreatment{X: item.Expr, Mode: mode}
				skipExpressionSpace(cursor)
			}
		}
		if ordered {
			if match := cursor.MatchOne(orderDirectionMatcher); match.Code == orderDirection {
				item.Direction = match.Text(cursor)
			}
			skipExpressionSpace(cursor)
			pos := cursor.Pos
			if match := cursor.MatchAny(limitKeywordMatcher, separatorKeywordMatcher); match.Code == limitKeyword || match.Code == separatorKeyword {
				cursor.Pos = pos
				return nil
			}
		} else if match := cursor.MatchOne(orderByKeywordMatcher); match.Code == orderByKeyword {
			op := match.Text(cursor)
			var ordering query.List
			if err := parseArgumentList(cursor, &ordering, true, false); err != nil {
				return err
			}
			item.Expr = &expr.Binary{X: item.Expr, Op: op, Y: ordering}
		}
		if !ordered {
			skipExpressionSpace(cursor)
			if match := cursor.MatchOne(separatorKeywordMatcher); match.Code == separatorKeyword {
				op := match.Text(cursor)
				skipExpressionSpace(cursor)
				separator, err := TryParseLiteral(cursor)
				if err != nil {
					return err
				}
				if separator == nil || separator.Kind != "string" {
					return cursor.NewError(singleQuotedStringLiteralMatcher, doubleQuotedStringLiteralMatcher)
				}
				item.Expr = &expr.Binary{X: item.Expr, Op: op, Y: separator}
				hasSeparator = true
				skipExpressionSpace(cursor)
			}
			if match := cursor.MatchOne(limitKeywordMatcher); match.Code == limitKeyword {
				op := match.Text(cursor)
				limit, err := expectExpression(cursor)
				if err != nil {
					return err
				}
				item.Expr = &expr.Binary{X: item.Expr, Op: op, Y: limit}
			}
		}
		skipExpressionSpace(cursor)
		if cursor.Pos == len(cursor.Input) {
			return nil
		}
		if hasSeparator {
			return fmt.Errorf("unexpected input after aggregate SEPARATOR at byte %d", cursor.Pos)
		}
		if cursor.MatchOne(nextMatcher).Code != nextCode {
			return cursor.NewError(nextMatcher)
		}
		// The next iteration requires an operand, including after a final comma.
	}
}

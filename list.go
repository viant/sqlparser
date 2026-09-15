package sqlparser

import (
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func parseSelectListItem(cursor *parsly.Cursor, list *query.List) error {
	operand, err := expectOperand(cursor)
	if operand == nil {
		return err
	}
	item := query.NewItem(operand)
	aliasStart := cursor.Pos
	if item.Alias, err = discoverAlias(cursor); err != nil {
		return err
	}
	aliasEnd := cursor.Pos
	list.Append(item)
	for {
		match := cursor.MatchAfterOptional(whitespaceMatcher, inlineCommentMatcher, commentBlockMatcher, binaryOperatorMatcher, logicalOperatorMatcher, nextMatcher)
		switch match.Code {
		case commentBlock:
			if item.Comments != "" {
				item.Comments += " "
			}
			item.Comments += match.Text(cursor)
			if item.Alias == "" {
				aliasStart = cursor.Pos
				if item.Alias, err = discoverAlias(cursor); err != nil {
					return err
				}
				aliasEnd = cursor.Pos
			}
		case logicalOperator, binaryOperator:
			cursor.Pos = match.Offset
			// An alias completes its expression; an operator cannot start a
			// second expression after it and silently replace that alias.
			if item.Alias != "" && !selectListBoundary(cursor, item, cursor.Input[aliasStart:aliasEnd]) {
				return cursor.NewError(nextMatcher, fromKeywordMatcher)
			}
			binaryExpr := expr.NewBinary(item.Expr)
			item.Expr = binaryExpr
			if err := parseBinaryExpr(cursor, binaryExpr); err != nil {
				return err
			}
			aliasStart = cursor.Pos
			if item.Alias, err = discoverAlias(cursor); err != nil {
				return err
			}
			aliasEnd = cursor.Pos
		case nextCode:
			count := len(*list)
			if err = parseSelectListItem(cursor, list); err != nil {
				return err
			}
			if count == len(*list) {
				return cursor.NewError(exprMatcher)
			}
			return nil
		default:
			if item.Alias == "" || selectListBoundary(cursor, item, cursor.Input[aliasStart:aliasEnd]) {
				return nil
			}
			err := cursor.NewError(nextMatcher, fromKeywordMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
			if cursor.OnError == nil {
				return err
			}
			pos := cursor.Pos
			if err := cursor.OnError(err, cursor, item); err != nil {
				return err
			}
			// Extensions must consume their own syntax. Recheck the boundary
			// afterwards so a nil error alone cannot discard a second alias.
			if cursor.Pos <= pos || cursor.Pos > len(cursor.Input) {
				return err
			}
		}
	}
}

// selectListBoundary only looks ahead at a completed projection. Clause bodies
// and template/dialect extensions remain with their existing parsing owners.
func selectListBoundary(cursor *parsly.Cursor, item *query.Item, aliasSyntax []byte) bool {
	if cursor.Pos == len(cursor.Input) || cursor.Input[cursor.Pos] == ';' {
		return true
	}
	// A native call may end before an opaque dialect suffix. An attached
	// bracket suffix or OVER (...) was previously retained in enclosing raw
	// CTE/subquery text. Neither is a completed implicit projection alias.
	// Explicit AS and separated bracket aliases do not take this path.
	if _, ok := item.Expr.(*expr.Call); ok {
		if len(aliasSyntax) > 0 && aliasSyntax[0] == '[' {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(string(aliasSyntax)), "OVER") && cursor.Input[cursor.Pos] == '(' {
			return true
		}
	}
	pos := cursor.Pos
	defer func() { cursor.Pos = pos }()
	identifierSize := selectorMatcher.Match(cursor)
	match := cursor.MatchAny(fromKeywordMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher, exceptKeywordMatcher)
	// The keyword matchers can recognize prefixes such as LIMIT in
	// limit_value. A longer identifier is not a clause boundary.
	if match.Size == 0 || match.Size < identifierSize {
		return false
	}
	return (aliasIdentifier{}).boundary(cursor) || strings.ContainsRune("(\"`[", rune(cursor.Input[cursor.Pos]))
}

func parseCallArgs(cursor *parsly.Cursor, list *query.List) error {
	return parseArgumentList(cursor, list, false)
}

func parseOrderByListItem(cursor *parsly.Cursor, list *query.List) error {

	operand, err := expectOperand(cursor)
	if operand == nil {
		return err
	}
	item := query.NewItem(operand)
	if matched := cursor.MatchAfterOptional(whitespaceMatcher, orderDirectionMatcher); matched.Code == orderDirection {
		item.Direction = matched.Text(cursor)
	}
	list.Append(item)
	match := cursor.MatchAfterOptional(whitespaceMatcher, inlineCommentMatcher, commentBlockMatcher, binaryOperatorMatcher, logicalOperatorMatcher, nextMatcher)
	switch match.Code {
	case commentBlock:
		item.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code == nextCode {
			return parseOrderByListItem(cursor, list)
		}
	case logicalOperator, binaryOperator:
		cursor.Pos -= match.Size
		binaryExpr := expr.NewBinary(item.Expr)
		item.Expr = binaryExpr
		if err := parseBinaryExpr(cursor, binaryExpr); err != nil {
			return err
		}
		if item.Alias, err = discoverAlias(cursor); err != nil {
			return err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			return nil
		}
		fallthrough
	case nextCode:
		return parseOrderByListItem(cursor, list)
	}
	return nil
}

func parseGroupByList(cursor *parsly.Cursor, list *query.List) error {
	operand, err := expectOperand(cursor)
	if operand == nil {
		return err
	}
	item := query.NewItem(operand)
	if matched := cursor.MatchAfterOptional(whitespaceMatcher, orderDirectionMatcher); matched.Code == orderDirection {
		item.Direction = matched.Text(cursor)
	}
	list.Append(item)
	match := cursor.MatchAfterOptional(whitespaceMatcher, inlineCommentMatcher, commentBlockMatcher, binaryOperatorMatcher, logicalOperatorMatcher, nextMatcher)
	switch match.Code {
	case commentBlock:
		item.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code == nextCode {
			return parseGroupByList(cursor, list)
		}
	case logicalOperator, binaryOperator:
		cursor.Pos -= match.Size
		binaryExpr := expr.NewBinary(item.Expr)
		item.Expr = binaryExpr
		if err := parseBinaryExpr(cursor, binaryExpr); err != nil {
			return err
		}
		if item.Alias, err = discoverAlias(cursor); err != nil {
			return err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			return nil
		}
		fallthrough
	case nextCode:
		return parseGroupByList(cursor, list)
	}
	return nil
}

// ParseList parses list
func ParseList(raw string) (query.List, error) {
	cursor := parsly.NewCursor("", []byte(raw), 0)
	list := query.List{}
	return list, parseSelectListItem(cursor, &list)
}

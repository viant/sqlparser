package sqlparser

import (
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
	item.Alias = discoverAlias(cursor)
	if match := cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher); match.Code == commentBlock {
		item.Comments = match.Text(cursor)
	}
	list.Append(item)
	match := cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher, inlineCommentMatcher, commentBlockMatcher, binaryOperatorMatcher, logicalOperatorMatcher, nextMatcher)
	switch match.Code {
	case commentBlock:
		item.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code == nextCode {
			return parseSelectListItem(cursor, list)
		}
	case logicalOperator, binaryOperator:
		cursor.Pos -= match.Size
		binaryExpr := expr.NewBinary(item.Expr)
		item.Expr = binaryExpr
		if err := parseBinaryExpr(cursor, binaryExpr); err != nil {
			return err
		}
		item.Alias = discoverAlias(cursor)
		if match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher); match.Code == commentBlock {
			item.Comments = match.Text(cursor)
		}

		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			return nil
		}
		fallthrough
	case nextCode:
		count := len(*list)
		if err = parseSelectListItem(cursor, list); err != nil {
			return err
		}
		if count == len(*list) {
			return cursor.NewError(exprMatcher)
		}
	}
	return nil
}

func parseCallArgs(cursor *parsly.Cursor, list *query.List) error {

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
			return parseCallArgs(cursor, list)
		}
	case logicalOperator, binaryOperator:
		cursor.Pos -= match.Size
		binaryExpr := expr.NewBinary(item.Expr)
		item.Expr = binaryExpr
		if err := parseBinaryExpr(cursor, binaryExpr); err != nil {
			return err
		}
		item.Alias = discoverAlias(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			return nil
		}
		fallthrough
	case nextCode:
		return parseCallArgs(cursor, list)
	}
	return nil
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
		item.Alias = discoverAlias(cursor)
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
		item.Alias = discoverAlias(cursor)
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

package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"strings"
)

func discoverAlias(cursor *parsly.Cursor) string {
	pos := cursor.Pos
	match := cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher, asKeywordMatcher, onKeywordMatcher, fromKeywordMatcher, joinToken, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, identifierMatcher)
	switch match.Code {
	case asKeyword:
		match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		return match.Text(cursor)
	case identifierCode:
		return match.Text(cursor)
	case exceptKeyword, fromKeyword, onKeyword, orderByKeyword, joinTokenCode, whereKeyword, groupByKeyword, havingKeyword, windowTokenCode:
		cursor.Pos = pos
	}
	return ""
}

func expectOperand(cursor *parsly.Cursor) (node.Node, error) {
	literal, err := TryParseLiteral(cursor)
	if literal != nil || err != nil {
		return literal, err
	}

	match := cursor.MatchAfterOptional(whitespaceMatcher,
		orderByKeywordMatcher,
		asKeywordMatcher,
		exceptKeywordMatcher,
		onKeywordMatcher, fromKeywordMatcher, whereKeywordMatcher, joinToken, groupByMatcher, havingKeywordMatcher, windowMatcher, nextMatcher,
		parenthesesMatcher,
		caseBlockMatcher,
		starTokenMatcher,
		notOperatorMatcher,
		nullMatcher,
		placeholderMatcher,
		selectorMatcher,
		commentBlockMatcher,
	)
	pos := cursor.Pos
	switch match.Code {
	case selectorTokenCode, placeholderTokenCode:

		selRaw := match.Text(cursor)
		var selector node.Node
		selector = expr.NewSelector(selRaw)
		if match.Code == placeholderTokenCode {
			selector = expr.NewPlaceholder(selRaw)
		}

		pos := cursor.Pos
		match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, exceptKeywordMatcher)
		switch match.Code {
		case parenthesesCode:
			raw := match.Text(cursor)
			var args []node.Node
			if len(raw) > 0 {
				argCursor := parsly.NewCursor(cursor.Path, []byte(raw[1:len(raw)-1]), pos)
				list := query.List{}
				if err := parseOrderByListItem(argCursor, &list); err != nil {
					return nil, err
				}
				for i := range list {
					args = append(args, list[i].Expr)
				}
			}
			return &expr.Call{X: selector, Raw: raw, Args: args}, nil

		case exceptKeyword:
			return parseStarExpr(cursor, selRaw, selector)
		}
		if strings.HasSuffix(selRaw, "*") {
			comments := ""
			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				comments = match.Text(cursor)
			}
			return expr.NewStar(selector, comments), nil
		}
		return selector, nil
	case exceptKeyword:
		return nil, cursor.NewError(selectorMatcher)
	case nullTokenCode:
		return expr.NewNullLiteral(match.Text(cursor)), nil
	case caseBlock:
		return &expr.Switch{Raw: match.Text(cursor)}, nil
	case starTokenCode:
		selRaw := match.Text(cursor)
		selector := expr.NewSelector(selRaw)
		match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
		comments := ""
		if match.Code == commentBlock {
			comments = match.Text(cursor)
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher)
		switch match.Code {
		case exceptKeyword:
			return parseStarExpr(cursor, selRaw, selector)
		}
		return expr.NewStar(selector, comments), err
	case parenthesesCode:
		return expr.NewParenthesis(match.Text(cursor)), nil
	case notOperator:
		unary := expr.NewUnary(match.Text(cursor))
		if unary.X, err = expectOperand(cursor); unary.X == nil || err != nil {
			return nil, cursor.NewError(selectorMatcher)
		}
		return unary, nil

	case asKeyword, orderByKeyword, onKeyword, fromKeyword, whereKeyword, joinTokenCode, groupByKeyword, havingKeyword, windowTokenCode, nextCode, commentBlock:
		cursor.Pos -= pos
	}
	return nil, nil
}

func parseStarExpr(cursor *parsly.Cursor, selRaw string, selector node.Node) (node.Node, error) {
	star := expr.NewStar(selector, "")
	if !strings.HasSuffix(selRaw, "*") {
		return star, nil
	}
	_, err := expectExpectIdentifiers(cursor, &star.Except)
	match := cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
	if match.Code == commentBlock {
		star.Comments = match.Text(cursor)
	}
	return star, err
}

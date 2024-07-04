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
	match := cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher, asKeywordMatcher, onKeywordMatcher, fromKeywordMatcher, joinMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher, identifierMatcher)
	switch match.Code {
	case asKeyword:
		match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		return match.Text(cursor)
	case identifierCode:
		return match.Text(cursor)
	case exceptKeyword, fromKeyword, onKeyword, orderByKeyword, joinToken, whereKeyword, groupByKeyword, havingKeyword, windowTokenCode, unionKeyword:
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
		onKeywordMatcher, fromKeywordMatcher, whereKeywordMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, windowMatcher, nextMatcher,
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
			args, err := parseCallArguments(cursor, raw, pos)
			if err != nil {
				return nil, err
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
		raw := match.Text(cursor)
		result := expr.NewParenthesis(raw)
		rawExpr := raw[1 : len(raw)-1]
		exprCursor := parsly.NewCursor(cursor.Path, []byte(rawExpr), cursor.Pos-len(raw))
		exprCursor.OnError = cursor.OnError
		binary := &expr.Binary{}
		_ = parseBinaryExpr(exprCursor, binary)
		result.X = result.X
		if binary.Y != nil {
			result.X = binary
		} else {
			exprCursor := parsly.NewCursor(cursor.Path, []byte(rawExpr), cursor.Pos-len(raw))

			var list []node.Node
			tokens := append([]*parsly.Token{placeholderMatcher}, literalTokens...)
			for i := 0; i < len(rawExpr); i++ {
				matched := exprCursor.MatchAfterOptional(whitespaceMatcher, tokens...)
				switch matched.Code {
				case nextCode:
				case placeholderTokenCode:
					list = append(list, &expr.Placeholder{Name: matched.Text(exprCursor)})
				case nullKeyword:
					list = append(list, expr.NewNullLiteral(matched.Text(exprCursor)))
				case singleQuotedStringLiteral, doubleQuotedStringLiteral:
					list = append(list, expr.NewStringLiteral(matched.Text(exprCursor)))
				case boolLiteral:
					list = append(list, expr.NewBoolLiteral(matched.Text(exprCursor)))
				case intLiteral:
					list = append(list, expr.NewIntLiteral(matched.Text(exprCursor)))
				case numericLiteral:
					list = append(list, expr.NewNumericLiteral(matched.Text(exprCursor)))
				default:
					break
				}
			}
			if len(list) > 0 {
				result.X = list
			}

		}
		return result, nil
	case notOperator:
		unary := expr.NewUnary(match.Text(cursor))
		if unary.X, err = expectOperand(cursor); unary.X == nil || err != nil {
			return nil, cursor.NewError(selectorMatcher)
		}
		return unary, nil
	case asKeyword, orderByKeyword, onKeyword, fromKeyword, whereKeyword, joinToken, groupByKeyword, havingKeyword, windowTokenCode, nextCode, commentBlock:
		cursor.Pos = pos
	}
	return nil, nil
}

func parseCallArguments(cursor *parsly.Cursor, raw string, pos int) ([]node.Node, error) {
	var args []node.Node
	if len(raw) > 0 {
		argCursor := parsly.NewCursor(cursor.Path, []byte(raw[1:len(raw)-1]), pos)
		argCursor.OnError = cursor.OnError
		list := query.List{}
		if err := parseCallArgs(argCursor, &list); err != nil {
			return nil, err
		}
		for i := range list {
			args = append(args, list[i].Expr)
		}
	}
	return args, nil
}

// ParseCallExpr parses call expression
func ParseCallExpr(rawExpr string) (*expr.Call, error) {
	cursor := parsly.NewCursor("", []byte(rawExpr), 0)
	match := cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return nil, cursor.NewError(selectorMatcher)
	}
	selector := expr.NewSelector(match.Text(cursor))
	pos := cursor.Pos
	match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
	if match.Code != parenthesesCode {
		return nil, cursor.NewError(parenthesesMatcher)
	}
	raw := match.Text(cursor)
	args, err := parseCallArguments(cursor, raw, pos)
	if err != nil {
		return nil, err
	}
	return &expr.Call{X: selector, Raw: rawExpr, Args: args}, nil
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

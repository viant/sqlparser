package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	smatcher "github.com/viant/sqlparser/matcher"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/source"
	"strings"
)

func expectOperand(cursor *parsly.Cursor) (node.Node, error) {
	operand, err := expectOperandBase(cursor)
	if err != nil || operand == nil {
		return operand, err
	}
	for {
		tail := *cursor
		skipExpressionSpace(&tail)
		if tail.Pos < len(tail.Input) && tail.Input[tail.Pos] == '.' {
			cursor.Pos = tail.Pos + 1
			skipExpressionSpace(cursor)
			name, err := expectFieldName(cursor)
			if err != nil {
				return nil, err
			}
			operand = &expr.FieldAccess{X: operand, Name: name}
			operand, err = applyCollate(cursor, operand)
			if err != nil {
				return nil, err
			}
			continue
		}
		// A separated [name] is a bracket-quoted alias in supported SQL
		// dialects. Only an adjacent bracket starts postfix element access.
		if cursor.Pos >= len(cursor.Input) || cursor.Input[cursor.Pos] != '[' || cursor.Pos > 0 && source.IsWhitespace(cursor.Input[cursor.Pos-1]) {
			return operand, nil
		}
		cursor.Pos++
		index, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		skipExpressionSpace(cursor)
		if cursor.Pos >= len(cursor.Input) || cursor.Input[cursor.Pos] != ']' {
			return nil, fmt.Errorf("expected closing subscript ']' at byte %d", cursor.Pos)
		}
		cursor.Pos++
		operand = &expr.Subscript{X: operand, Index: index}
		operand, err = applyCollate(cursor, operand)
		if err != nil {
			return nil, err
		}
	}
}

func expectFieldName(cursor *parsly.Cursor) (string, error) {
	start := cursor.Pos
	if start < len(cursor.Input) {
		first := cursor.Input[start]
		if first == '`' || first == '"' || first == '[' {
			end := first
			if first == '[' {
				end = ']'
			}
			cursor.Pos++
			for cursor.Pos < len(cursor.Input) {
				c := cursor.Input[cursor.Pos]
				cursor.Pos++
				// Backslash escapes quoted text, but is literal in bracketed
				// identifiers, matching the source scanner's quote handling.
				if c == '\\' && end != ']' {
					if cursor.Pos == len(cursor.Input) {
						break
					}
					cursor.Pos++
					continue
				}
				if c != end {
					continue
				}
				if cursor.Pos < len(cursor.Input) && cursor.Input[cursor.Pos] == end {
					cursor.Pos++
					continue
				}
				if cursor.Pos > start+2 {
					return string(cursor.Input[start:cursor.Pos]), nil
				}
				break
			}
		} else if smatcher.IsLetter(first) || first == '_' {
			cursor.Pos++
			for cursor.Pos < len(cursor.Input) {
				c := cursor.Input[cursor.Pos]
				if !smatcher.IsLetter(c) && c != '_' && !(c >= '0' && c <= '9') {
					break
				}
				cursor.Pos++
			}
			return string(cursor.Input[start:cursor.Pos]), nil
		}
	}
	return "", fmt.Errorf("expected field name at byte %d", start)
}

func expectOperandBase(cursor *parsly.Cursor) (node.Node, error) {
	literal, err := TryParseLiteral(cursor)
	if literal != nil || err != nil {
		if err != nil {
			return literal, err
		}
		return applyCollate(cursor, literal)
	}

	match := cursor.MatchAfterOptional(whitespaceMatcher,
		whenKeywordMatcher, thenKeywordMatcher, elseKeywordMatcher, endKeywordMatcher,
		intervalKeywordMatcher,
		orderByKeywordMatcher,
		asKeywordMatcher,
		exceptKeywordMatcher,
		onKeywordMatcher, fromKeywordMatcher, whereKeywordMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, windowMatcher, nextMatcher,
		parenthesesMatcher,
		caseBlockMatcher,
		starTokenMatcher,
		notOperatorMatcher,
		bitwiseNotMatcher,
		nullMatcher,
		placeholderMatcher,
		selectorMatcher,
		commentBlockMatcher,
	)
	pos := cursor.Pos
	// OFFSET is also a pagination keyword, but OFFSET(...) in an operand
	// position is a function (not a query clause).
	if match.Code == windowTokenCode && strings.EqualFold(match.Text(cursor), "OFFSET") {
		if call := matchCallParentheses(cursor); call.Code == parenthesesCode {
			raw := call.Text(cursor)
			args, err := parseCallArguments(cursor, "OFFSET", raw, pos)
			if err != nil {
				return nil, err
			}
			return applyCollate(cursor, &expr.Call{X: expr.NewSelector("OFFSET"), Raw: raw, Args: args})
		}
		cursor.Pos = pos
	}

	switch match.Code {
	case selectorTokenCode, placeholderTokenCode:

		selRaw := match.Text(cursor)
		var selector node.Node
		selector = expr.NewSelector(selRaw)
		if match.Code == placeholderTokenCode {
			selector = expr.NewPlaceholder(selRaw)
		}

		pos := cursor.Pos
		match = matchCallParentheses(cursor)
		if match.Code == parenthesesCode {
			raw := match.Text(cursor)
			args, err := parseCallArguments(cursor, selRaw, raw, pos)
			if err != nil {
				return nil, err
			}
			return applyCollate(cursor, &expr.Call{X: selector, Raw: raw, Args: args})
		}
		if match = cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher); match.Code == exceptKeyword {
			return parseStarExpr(cursor, selRaw, selector)
		}
		if strings.HasSuffix(selRaw, "*") {
			comments := ""
			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				comments = match.Text(cursor)
			}
			return applyCollate(cursor, expr.NewStar(selector, comments))
		}
		return applyCollate(cursor, selector)
	case exceptKeyword:
		return nil, cursor.NewError(selectorMatcher)
	case nullTokenCode:
		return applyCollate(cursor, expr.NewNullLiteral(match.Text(cursor)))
	case caseBlock:
		result, err := parseCase(cursor, cursor.Pos-match.Size)
		if err != nil {
			return nil, err
		}
		return applyCollate(cursor, result)
	case intervalKeyword:
		op := match.Text(cursor)
		value, err := expectOperand(cursor)
		if err != nil || value == nil {
			return nil, cursor.NewError(exprMatcher)
		}
		interval := &expr.Unary{Op: op, X: value}
		skipExpressionSpace(cursor)
		if unit := cursor.MatchOne(intervalUnitMatcher); unit.Code == intervalUnit {
			interval.X = &expr.Binary{X: value, Y: &expr.Ident{Name: unit.Text(cursor)}}
		} else if literal, ok := value.(*expr.Literal); !ok || literal.Kind != "string" {
			return nil, cursor.NewError(intervalUnitMatcher)
		}
		return applyCollate(cursor, interval)
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
		return applyCollate(cursor, expr.NewStar(selector, comments))
	case parenthesesCode:
		raw := match.Text(cursor)
		result := expr.NewParenthesis(raw)
		rawExpr := raw[1 : len(raw)-1]
		exprCursor := parsly.NewCursor(cursor.Path, []byte(rawExpr), cursor.Pos-len(raw))
		exprCursor.OnError = cursor.OnError
		skipExpressionSpace(exprCursor)
		start := exprCursor.Pos
		if match := exprCursor.MatchAny(selectKeywordMatcher, withKeywordMatcher); match.Code == selectKeyword || match.Code == withKeyword {
			exprCursor.Pos = start
			selectNode := &query.Select{}
			if err := parseQuery(exprCursor, selectNode); err != nil {
				return nil, err
			}
			skipExpressionSpace(exprCursor)
			if exprCursor.Pos != len(exprCursor.Input) {
				return nil, exprCursor.NewError(exprMatcher)
			}
			result.X = selectNode
			return applyCollate(cursor, result)
		}
		var list query.List
		if err := parseCallArgs(exprCursor, &list); err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, exprCursor.NewError(exprMatcher)
		}
		if len(list) == 1 {
			result.X = list[0].Expr
		} else {
			items := make([]node.Node, len(list))
			for i := range list {
				items[i] = list[i].Expr
			}
			result.X = items
		}
		return applyCollate(cursor, result)
	case notOperator:
		unary := expr.NewUnary(match.Text(cursor))
		if unary.X, err = expectOperand(cursor); unary.X == nil || err != nil {
			return nil, cursor.NewError(selectorMatcher)
		}
		return applyCollate(cursor, unary)
	case commentBlock:
		return expectOperand(cursor)
	case whenKeyword, thenKeyword, elseKeyword, endKeyword, asKeyword, orderByKeyword, onKeyword, fromKeyword, whereKeyword, joinToken, groupByKeyword, havingKeyword, windowTokenCode, nextCode:
		cursor.Pos = pos - match.Size
	}
	if match.Code == parsly.Invalid && cursor.OnError != nil {
		start := cursor.Pos
		var operand node.Node
		parseErr := cursor.NewError(exprMatcher)
		if err := cursor.OnError(parseErr, cursor, &operand); err != nil {
			return nil, err
		}
		if operand == nil || cursor.Pos <= start || cursor.Pos > len(cursor.Input) {
			return nil, parseErr
		}
		return applyCollate(cursor, operand)
	}
	return nil, nil
}

func applyCollate(cursor *parsly.Cursor, n node.Node) (node.Node, error) {
	if n == nil {
		return nil, nil
	}
	pos := cursor.Pos
	match := cursor.MatchAfterOptional(whitespaceMatcher, collateKeywordMatcher)
	if match.Code != collateKeyword {
		cursor.Pos = pos
		return n, nil
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	if match.Code != identifierCode {
		return nil, cursor.NewError(identifierMatcher)
	}
	return &expr.Collate{X: n, Collation: match.Text(cursor)}, nil
}

func parseCallArguments(cursor *parsly.Cursor, name, raw string, pos int) ([]node.Node, error) {
	var args []node.Node
	if len(raw) > 0 {
		if strings.EqualFold(name, "extract") {
			return parseExtractArguments(cursor, raw, pos)
		}
		if strings.EqualFold(name, "cast") {
			if index := source.FindTopLevelKeyword(raw[1:len(raw)-1], "AS", 0); index >= 0 {
				return parseCastArguments(cursor, raw, pos, index)
			}
		}
		argCursor := parsly.NewCursor(cursor.Path, []byte(raw[1:len(raw)-1]), pos)
		argCursor.OnError = cursor.OnError
		if strings.EqualFold(name, "struct") {
			return parseStructArguments(argCursor)
		}
		// Query arguments retain their own scope in the AST. ARRAY also has
		// scalar forms in other dialects, so recognize its query form first.
		if strings.EqualFold(name, "exists") || strings.EqualFold(name, "array") && startsQueryArgument(argCursor) {
			queryNode, err := parseQueryArgument(argCursor)
			if err != nil {
				return nil, err
			}
			return []node.Node{queryNode}, nil
		}
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

// Comments may separate a function name from its arguments. Leave the cursor
// unchanged when there is no call so aliases and projection comments keep
// their existing parsing owners.
func matchCallParentheses(cursor *parsly.Cursor) *parsly.TokenMatch {
	pos := cursor.Pos
	skipExpressionSpace(cursor)
	match := cursor.MatchOne(parenthesesMatcher)
	if match.Code != parenthesesCode {
		cursor.Pos = pos
	}
	return match
}

// ParseCallExpr parses call expression
func ParseCallExpr(rawExpr string) (*expr.Call, error) {
	if err := source.ValidateStructure(rawExpr); err != nil {
		return nil, err
	}
	cursor := parsly.NewCursor("", []byte(rawExpr), 0)
	match := cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return nil, cursor.NewError(selectorMatcher)
	}
	selector := expr.NewSelector(match.Text(cursor))
	pos := cursor.Pos
	match = matchCallParentheses(cursor)
	if match.Code != parenthesesCode {
		return nil, cursor.NewError(parenthesesMatcher)
	}
	raw := match.Text(cursor)
	args, err := parseCallArguments(cursor, Stringify(selector), raw, pos)
	if err != nil {
		return nil, err
	}
	skipExpressionSpace(cursor)
	if cursor.Pos != len(cursor.Input) {
		return nil, cursor.NewError(exprMatcher)
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

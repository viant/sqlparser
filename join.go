package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"strings"
)

func parseJoin(cursor *parsly.Cursor, join *query.Join, dest *query.Select, expectOn bool) error {
	defer func() {
		if join.Span.End == 0 {
			join.Span.End = uint32(cursor.Pos)
		}
	}()
	if err := parseJoinTarget(cursor, join); err != nil {
		return err
	}
	if join.With == nil {
		return cursor.NewError(parenthesesMatcher, selectorMatcher)
	}
	if join.Alias == "" {
		var err error
		if join.Alias, err = discoverAlias(cursor); err != nil {
			return err
		}
	}
	match := cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher, onKeywordMatcher)
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, onKeywordMatcher)
	}
	switch match.Code {
	case onKeyword:
		begin := match.Offset
		binary := &expr.Binary{}
		join.On = &expr.Qualify{}
		join.On.X = binary
		if err := parseBinaryExpr(cursor, binary); err != nil {
			return err
		}
		raw := string(cursor.Input[begin:cursor.Pos])
		trimmed := strings.TrimSpace(raw)
		start := begin + len(raw) - len(strings.TrimLeft(raw, " \t\r\n"))
		join.OnSpan = node.Span{Begin: uint32(start), End: uint32(start + len(trimmed))}
	default:
		if expectOn {
			return cursor.NewError(onKeywordMatcher)
		}
	}
	join.Span.End = uint32(len(strings.TrimRight(string(cursor.Input[:cursor.Pos]), " \t\r\n")))
	match = cursor.MatchAfterOptional(whitespaceMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
	if match.Code == parsly.EOF {
		return nil
	}
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
		if match.Code == parsly.EOF {
			return nil
		}
	}

	hasMatch, err := matchPostFrom(cursor, dest, match)
	if !hasMatch && err == nil {
		err = cursor.NewError(joinMatcher, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
	}

	return err
}

func parseJoinTarget(cursor *parsly.Cursor, join *query.Join) error {
	pos := cursor.Pos
	if match := cursor.MatchAfterOptional(whitespaceMatcher, bracedTableMatcher); match.Code == tableTokenCode {
		join.With = expr.NewSelector(match.Text(cursor))
		return nil
	}
	cursor.Pos = pos
	operand, err := expectOperand(cursor)
	if err != nil {
		return err
	}
	if operand != nil {
		join.With = operand
		return nil
	}
	cursor.Pos = pos

	match := cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, exprMatcher, selectorMatcher)
	switch match.Code {
	case parenthesesCode:
		join.With = expr.NewRaw(match.Text(cursor))
	case selectorTokenCode:
		identityOrAlias := match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
		if match.Code == parenthesesCode {
			identityOrAlias += match.Text(cursor)
		}
		join.With = expr.NewSelector(identityOrAlias)
	}
	return nil
}

func parseDeleteJoin(cursor *parsly.Cursor, join *query.Join) (*parsly.TokenMatch, error) {
	match := cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, selectorMatcher)
	switch match.Code {
	case parenthesesCode:
		join.With = expr.NewRaw(match.Text(cursor))
	case selectorTokenCode:
		join.With = expr.NewSelector(match.Text(cursor))
	}

	var err error
	if join.Alias, err = discoverAlias(cursor); err != nil {
		return match, err
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher, onKeywordMatcher)
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, onKeywordMatcher)
	}
	switch match.Code {
	case onKeyword:
	default:
		return match, cursor.NewError(onKeywordMatcher)
	}
	binary := &expr.Binary{}
	join.On = &expr.Qualify{}
	join.On.X = binary
	if err := parseBinaryExpr(cursor, binary); err != nil {
		return match, err
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher)
	if match.Code == parsly.EOF {
		return match, nil
	}
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, joinMatcher, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher)
		if match.Code == parsly.EOF {
			return match, nil
		}
	}
	return match, nil
}

func appendJoin(cursor *parsly.Cursor, match *parsly.TokenMatch, dest *query.Select, expectOn bool) error {
	join := query.NewJoin(match.Text(cursor))
	join.Span.Begin = uint32(match.Offset)

	dest.Joins = append(dest.Joins, join)
	if err := parseJoin(cursor, join, dest, expectOn); err != nil {
		return err
	}
	return nil
}

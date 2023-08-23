package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func parseJoin(cursor *parsly.Cursor, join *query.Join, dest *query.Select, expectOn bool) error {
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

		withSelect := dest.WithSelects.Select(identityOrAlias)
		if withSelect != nil {
			join.With = expr.NewParenthesis(withSelect.Raw)
			join.Alias = identityOrAlias
		} else {
			join.With = expr.NewSelector(identityOrAlias)
		}
	}

	if join.Alias == "" {
		join.Alias = discoverAlias(cursor)
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher, onKeywordMatcher)
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, onKeywordMatcher)
	}
	switch match.Code {
	case onKeyword:
		binary := &expr.Binary{}
		join.On = &expr.Qualify{}
		join.On.X = binary
		if err := parseBinaryExpr(cursor, binary); err != nil {
			return err
		}
	default:
		if expectOn {
			return cursor.NewError(onKeywordMatcher)
		}
	}
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

func parseDeleteJoin(cursor *parsly.Cursor, join *query.Join) (*parsly.TokenMatch, error) {
	match := cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher, selectorMatcher)
	switch match.Code {
	case parenthesesCode:
		join.With = expr.NewRaw(match.Text(cursor))
	case selectorTokenCode:
		join.With = expr.NewSelector(match.Text(cursor))
	}

	join.Alias = discoverAlias(cursor)

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

	dest.Joins = append(dest.Joins, join)
	if err := parseJoin(cursor, join, dest, expectOn); err != nil {
		return err
	}
	return nil
}

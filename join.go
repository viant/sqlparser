package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func parseJoins(cursor *parsly.Cursor, join *query.Join, dest *query.Select) error {
	match, err := parseJoin(cursor, join)
	if err != nil {
		return err
	}

	hasMatch, err := matchPostFrom(cursor, dest, match)
	if !hasMatch && err == nil {
		err = cursor.NewError(joinToken, joinToken, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher)
	}

	return err
}

func parseJoin(cursor *parsly.Cursor, join *query.Join) (*parsly.TokenMatch, error) {
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
	match = cursor.MatchAfterOptional(whitespaceMatcher, joinToken, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher)
	if match.Code == parsly.EOF {
		return match, nil
	}
	if match.Code == commentBlock {
		join.Comments = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, joinToken, groupByMatcher, havingKeywordMatcher, whereKeywordMatcher, orderByKeywordMatcher, windowMatcher)
		if match.Code == parsly.EOF {
			return match, nil
		}
	}
	return match, nil
}

func appendJoin(cursor *parsly.Cursor, match *parsly.TokenMatch, dest *query.Select) error {
	join := query.NewJoin(match.Text(cursor))
	dest.Joins = append(dest.Joins, join)
	if err := parseJoins(cursor, join, dest); err != nil {
		return err
	}
	return nil
}

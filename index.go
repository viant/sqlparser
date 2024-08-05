package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/index"
	"strings"
)

// ParseDropIndex parses drop table
func ParseDropIndex(SQL string) (*index.Drop, error) {
	result := &index.Drop{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseDropIndex(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%s", SQL)
	}
	return result, err
}

func parseDropIndex(cursor *parsly.Cursor, dest *index.Drop) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, dropIndexMatcher)
	if match.Code != dropIndexToken {
		return cursor.NewError(createMatcher)
	}
	if match = cursor.MatchOne(whitespaceMatcher); match.Code != whitespaceCode {
		return cursor.NewError(whitespaceMatcher)
	}
	if match = cursor.MatchOne(ifExistsMatcher); match.Code == ifExistsToken {
		dest.IfExists = true
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	if match.Code != identifierCode {
		return cursor.NewError(ifNotExistsMatcher)
	}
	dest.Name = match.Text(cursor)
	match = cursor.MatchAfterOptional(whitespaceMatcher, onKeywordMatcher)
	if match.Code != onKeyword {
		return cursor.NewError(ifNotExistsMatcher)
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code == selectorTokenCode {
		selector := match.Text(cursor)
		if index := strings.Index(selector, "."); index != -1 {
			dest.Schema = selector[0:index]
			dest.Table = selector[index+1:]
		} else {
			dest.Table = selector
		}
	}
	return nil
}

// ParseCreateIndex parses create table
func ParseCreateIndex(SQL string) (*index.Create, error) {
	result := &index.Create{}
	SQL = removeSQLComments(SQL)
	result.Spec.SQL = SQL
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseCreateIndex(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%s", SQL)
	}
	return result, err
}

func parseCreateIndex(cursor *parsly.Cursor, dest *index.Create) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, createMatcher)
	if match.Code != createToken {
		return cursor.NewError(createMatcher)
	}
	if match = cursor.MatchOne(whitespaceMatcher); match.Code != whitespaceCode {
		return cursor.NewError(whitespaceMatcher)
	}

	if match = cursor.MatchOne(ifNotExistsMatcher); match.Code == ifNotExistsToken {
		dest.IfDoesExists = true
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, indexMatcher)
	if match.Code == indexToken {
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(ifNotExistsMatcher)
		}
		dest.Spec.Name = match.Text(cursor)

	} else {
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(ifNotExistsMatcher)
		}
		dest.Spec.Type = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, indexMatcher)
		if match.Code != indexToken {
			return cursor.NewError(indexMatcher)
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(ifNotExistsMatcher)
		}
		dest.Spec.Name = match.Text(cursor)
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, onKeywordMatcher)
	if match.Code != onKeyword {
		return cursor.NewError(parenthesesMatcher)
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return cursor.NewError(selectorMatcher)

	}
	selector := match.Text(cursor)
	if index := strings.Index(selector, "."); index != -1 {
		dest.Spec.Schema = selector[0:index]
		dest.Spec.Table = selector[index+1:]
	} else {
		dest.Spec.Table = selector
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
	if match.Code != parenthesesCode {
		return cursor.NewError(parenthesesMatcher)
	}
	columnSpec := match.Text(cursor)
	pos := cursor.Pos
	columnCursor := parsly.NewCursor(cursor.Path, []byte(columnSpec[1:len(columnSpec)-1]), pos)

	return parseIndexColumnSpec(dest, match, columnCursor)
}

func parseIndexColumnSpec(dest *index.Create, match *parsly.TokenMatch, cursor *parsly.Cursor) error {
	for {
		col := &index.ColumnSpec{}
		dest.Spec.Columns = append(dest.Spec.Columns, col)
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(ifNotExistsMatcher)
		}
		col.Name = match.Text(cursor)
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			break
		}
	}
	return nil
}

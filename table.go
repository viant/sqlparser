package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/column"
	"github.com/viant/sqlparser/table"
)

func ParseCreateTable(SQL string) (*table.Create, error) {
	result := &table.Create{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseCreateTable(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%s", SQL)
	}
	return result, err
}

func parseCreateTable(cursor *parsly.Cursor, dest *table.Create) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, createTableMatcher)
	if match.Code != createTableToken {
		return cursor.NewError(createTableMatcher)
	}
	if match = cursor.MatchOne(whitespaceMatcher); match.Code != whitespaceCode {
		return cursor.NewError(whitespaceMatcher)
	}

	if match = cursor.MatchOne(ifNotExistsMatcher); match.Code == ifNotExistsToken {
		dest.IfDoesExists = true
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	if match.Code != identifierCode {
		return cursor.NewError(ifNotExistsMatcher)
	}
	dest.Spec.Name = match.Text(cursor)
	pos := cursor.Pos
	match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
	if match.Code != parenthesesCode {
		return cursor.NewError(parenthesesMatcher)
	}
	columnSpec := match.Text(cursor)
	columnCursor := parsly.NewCursor(cursor.Path, []byte(columnSpec[1:len(columnSpec)-1]), pos)
	return parseColumnSpec(dest, match, columnCursor)
}

func parseColumnSpec(dest *table.Create, match *parsly.TokenMatch, cursor *parsly.Cursor) error {
	for {
		col := &column.Spec{}
		dest.Spec.Columns = append(dest.Spec.Columns, col)
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(ifNotExistsMatcher)
		}
		col.Name = match.Text(cursor)
		node, err := expectOperand(cursor)
		if err != nil {
			return err
		}
		col.Type = Stringify(node)

		match = cursor.MatchAfterOptional(whitespaceMatcher, keyMatcher)
		if match.Code == keyTokenCode {
			col.Key = match.Text(cursor)
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, notNullMatcher)
		col.Nullable = match.Code != notNullToken

		match = cursor.MatchAfterOptional(whitespaceMatcher, defaultMatcher)
		if match.Code == defaultToken {
			node, err := expectOperand(cursor)
			if err != nil {
				return err
			}
			def := Stringify(node)
			col.Default = &def
		}

		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code != nextCode {
			break
		}
	}
	return nil
}

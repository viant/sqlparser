package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/column"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/table"
	"github.com/viant/sqlparser/update"
)

// TableName returns main table name
func TableName(node node.Node) string {
	switch actual := node.(type) {
	case *query.Select:
		return queryTableName(actual)
	case *insert.Statement:
		return trimEnclosure(actual.Target.X)
	case *update.Statement:
		return trimEnclosure(actual.Target.X)
	case *del.Statement:
		return trimEnclosure(actual.Target.X)
	case *table.Create:
		return trimEnclosure(actual.Spec.Name)
	case *table.Drop:
		return trimEnclosure(actual.Name)
	}
	return ""
}

// TableSelector returns the table selector Node
func TableSelector(n node.Node) *expr.Selector {
	switch actual := n.(type) {
	case *query.Select:
		return queryTableSelector(actual)
	case *insert.Statement:
		return extractSelector(actual.Target.X)
	case *update.Statement:
		return extractSelector(actual.Target.X)
	case *del.Statement:
		return extractSelector(actual.Target.X)
	case *table.Create:
		return extractSelector(actual.Spec.Name)
	case *table.Drop:
		return extractSelector(actual.Name)
	}
	return nil
}

func queryTableSelector(s *query.Select) *expr.Selector {
	if s.From.X == nil {
		return nil
	}
	return extractSelector(s.From.X)
}

func extractSelector(node node.Node) *expr.Selector {
	switch t := node.(type) {
	case *expr.Selector:
		return t
	case *expr.Ident:
		return &expr.Selector{Name: t.Name}
	}
	return nil
}

func queryTableName(sel *query.Select) string {
	if sel.From.X == nil {
		return ""
	}
	switch actual := sel.From.X.(type) {
	case *expr.Ident:
		return actual.Name
	case *expr.Selector:
		return trimEnclosure(actual)
	case *expr.Parenthesis:
		raw := trimEnclosure(actual.Raw)
		if sel, _ := ParseQuery(raw); sel != nil {
			actual.X = sel
			return TableName(sel)
		}
	case *expr.Raw:
		if actual.X != nil {
			return TableName(actual.X)
		}
	default:
		panic(fmt.Sprintf("not supported:%T ", actual))
	}
	return ""
}

func trimEnclosure(node node.Node) string {
	if node == nil {
		return ""
	}
	var name string
	var ok bool
	if name, ok = node.(string); !ok {
		name = Stringify(node)
	}
	switch name[0] {
	case '`', '"', '[', '\'', '(':
		name = name[1 : len(name)-1]
	}
	return name
}

// ParseTruncateTable parses truncate table
func ParseTruncateTable(SQL string) (*table.Truncate, error) {
	result := &table.Truncate{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseTruncateTable(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%s", SQL)
	}
	return result, err
}

func parseTruncateTable(cursor *parsly.Cursor, result *table.Truncate) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, truncateKeywordMatcher)
	if match.Code != truncateTableKeyword {
		return cursor.NewError(truncateKeywordMatcher)
	}
	if match = cursor.MatchOne(whitespaceMatcher); match.Code != whitespaceCode {
		return cursor.NewError(whitespaceMatcher)
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	if match.Code != identifierCode {
		return cursor.NewError(identifierMatcher)
	}
	result.Table = match.Text(cursor)
	return nil
}

// ParseDropTable parses drop table
func ParseDropTable(SQL string) (*table.Drop, error) {
	result := &table.Drop{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseDropTable(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%s", SQL)
	}
	return result, err
}

func parseDropTable(cursor *parsly.Cursor, dest *table.Drop) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, dropTableMatcher)
	if match.Code != dropTableToken {
		return cursor.NewError(createTableMatcher)
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
	return nil
}

// ParseCreateTable parses create table
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
		col.IsNullable = match.Code != notNullToken

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

package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/node"
	"strings"
)

// ParseInsert Parses INSERT INTO statement
func ParseInsert(SQL string) (*insert.Statement, error) {
	result := &insert.Statement{}
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	return result, parseInsert(cursor, result)
}

func parseInsert(cursor *parsly.Cursor, stmt *insert.Statement) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, insertIntoKeywordMatcher)
	switch match.Code {
	case insertIntoKeyword:
		match = cursor.MatchAfterOptional(whitespaceMatcher, tableMatcher)
		switch match.Code {
		case tableTokenCode:
			sel := match.Text(cursor)
			stmt.Target.X = expr.NewSelector(sel)
			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				stmt.Target.Comments = match.Text(cursor)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
			if match.Code == parenthesesCode {
				matched := match.Text(cursor)
				stmt.Columns = extractColumnNames(matched[1 : len(matched)-1])
			}

			match = cursor.MatchAfterOptional(whitespaceMatcher, insertValesKeywordMatcher)
			if match.Code != insertValuesKeyword {
				return cursor.NewError(insertValesKeywordMatcher)
			}
			offset := cursor.Pos
			match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
			if match.Code != parenthesesCode {
				return cursor.NewError(parenthesesMatcher)
			}
			matched := match.Text(cursor)
			var err error
			if stmt.Values, err = parseInsertValues(matched[1:len(matched)-1], offset); err != nil {
				return err
			}
			for i := cursor.Pos; i < len(cursor.Input); i++ {
				match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
				if match.Code != nextCode {
					break
				}
				match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
				if match.Code != parenthesesCode {
					return cursor.NewError(parenthesesMatcher)
				}
				values, err := parseInsertValues(matched[1:len(matched)-1], offset)
				if err != nil {
					return err
				}
				stmt.Values = append(stmt.Values, values...)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, asKeywordMatcher)
			if match.Code == asKeyword {
				match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
				if match.Code != selectorTokenCode {
					return cursor.NewError(selectorMatcher)
				}
				stmt.Alias = match.Text(cursor)
				if match = cursor.MatchAfterOptional(whitespaceMatcher, onDuplicateKeyUpdateMatcher); match.Code == onDuplicateKeyUpdate {
					item, err := expectUpdateSetItem(cursor)
					if err != nil {
						return err
					}
					stmt.OnDuplicateKeyUpdate = append(stmt.OnDuplicateKeyUpdate, item)
					if err = parseDuplicateSetItems(cursor, stmt); err != nil {
						return err
					}
				}

			}
		}
	default:
		return cursor.NewError(insertIntoKeywordMatcher)
	}
	return nil
}

func parseDuplicateSetItems(cursor *parsly.Cursor, stmt *insert.Statement) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		item, err := expectUpdateSetItem(cursor)
		if err != nil {
			return err
		}
		stmt.OnDuplicateKeyUpdate = append(stmt.OnDuplicateKeyUpdate, item)
		return parseDuplicateSetItems(cursor, stmt)
	case parsly.EOF:
	default:
		return nil
	}
	return nil
}

func parseInsertValues(encodedValues string, offset int) ([]*insert.Value, error) {
	cursor := parsly.NewCursor("", []byte(encodedValues), offset)
	var values []*insert.Value
	if err := expectInsertValue(cursor, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func expectInsertValue(cursor *parsly.Cursor, values *[]*insert.Value) error {
	pos := cursor.Pos
	operand, err := expectOperand(cursor)
	if err != nil || operand == nil {
		return err
	}
	*values = append(*values, &insert.Value{Expr: operand,
		Span: node.Span{Begin: uint32(pos), End: uint32(cursor.Pos)},
		Raw:  strings.TrimSpace(string(cursor.Input[pos:cursor.Pos]))})
	match := cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	if match.Code != nextCode {
		return nil
	}
	return expectInsertValue(cursor, values)
}

func extractColumnNames(separatedColumns string) []string {
	var result = make([]string, strings.Count(separatedColumns, ",")+1)
	var index = 0
	for _, column := range strings.Split(separatedColumns, ",") {
		column = strings.TrimSpace(column)
		if column == "" {
			continue
		}
		result[index] = column
		index++
	}
	return result[:index]
}

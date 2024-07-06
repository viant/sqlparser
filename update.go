package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/update"
)

// ParseUpdate parses update statement
func ParseUpdate(SQL string) (*update.Statement, error) {
	result := &update.Statement{}
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	return result, parseUpdate(cursor, result)

}

func parseUpdate(cursor *parsly.Cursor, stmt *update.Statement) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, updateKeywordMatcher)
	switch match.Code {
	case updateKeyword:
		match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
		switch match.Code {
		case selectorTokenCode:
			stmt.Target = extractTarget(cursor, match)
			match = cursor.MatchAfterOptional(whitespaceMatcher, setKeywordMatcher)
			if match.Code != setKeyword {
				return cursor.NewError(setKeywordMatcher)
			}

			item, err := expectUpdateSetItem(cursor)
			if err != nil {
				return err
			}
			stmt.Set = append(stmt.Set, item)
			if err = parseUpdateSetItems(cursor, stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractTarget(cursor *parsly.Cursor, match *parsly.TokenMatch) update.Target {
	sel := match.Text(cursor)
	comment := ""
	match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
	if match.Code == commentBlock {
		comment = match.Text(cursor)
	}

	return update.Target{
		X:        expr.NewSelector(sel),
		Comments: comment,
	}
}

func parseUpdateSetItems(cursor *parsly.Cursor, stmt *update.Statement) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, whereKeywordMatcher, nextMatcher)
	switch match.Code {
	case whereKeyword:
		stmt.Qualify = &expr.Qualify{}
		if err := ParseQualify(cursor, stmt.Qualify); err != nil {
			return err
		}
	case nextCode:
		item, err := expectUpdateSetItem(cursor)
		if err != nil {
			return err
		}
		stmt.Set = append(stmt.Set, item)
		return parseUpdateSetItems(cursor, stmt)
	case parsly.EOF:
	default:
		return cursor.NewError(nextMatcher, whereKeywordMatcher)
	}
	return nil
}

func expectUpdateSetItem(cursor *parsly.Cursor) (*update.Item, error) {
	beginPos := cursor.Pos
	match := cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return nil, cursor.NewError(selectorMatcher)
	}
	selRaw := match.Text(cursor)
	item := &update.Item{Column: expr.NewSelector(selRaw)}
	match = cursor.MatchAfterOptional(whitespaceMatcher, assignOperatorMatcher)
	if match.Code != assignOperator {
		return nil, cursor.NewError(assignOperatorMatcher)
	}
	operand, err := expectOperand(cursor)
	if err != nil {
		return nil, err
	}

	pos := cursor.Pos
	binary := &expr.Binary{}
	binary.X = operand
	if err = parseBinaryExpr(cursor, binary); err == nil {
		item.Expr = binary
	} else {
		cursor.Pos = pos
		item.Expr = operand
	}

	item.Begin = uint32(beginPos) + 1
	item.End = uint32(cursor.Pos)
	return item, err
}

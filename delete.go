package sqlparser

import (
	"github.com/viant/parsly"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func ParseDelete(SQL string) (*del.Statement, error) {
	aStmt := &del.Statement{}
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	return aStmt, parseDelete(aStmt, cursor)
}

func parseDelete(stmt *del.Statement, cursor *parsly.Cursor) error {
	matched := cursor.MatchAfterOptional(whitespaceMatcher, deleteMatcher)
	if matched.Code != deleteCode {
		return cursor.NewError(deleteMatcher)
	}

	var err error
	stmt.Items, err = tryParseDeleteItems(cursor)
	if err != nil {
		return err
	}

	lastMatchedCode, err := buildDeleteTarget(stmt, cursor)
	if err != nil {
		return err
	}

	stmt.Qualify, err = tryParseQualify(cursor, lastMatchedCode)
	if err != nil {
		return err
	}

	return nil
}

func buildDeleteTarget(stmt *del.Statement, cursor *parsly.Cursor) (int, error) {
	matchable := []*parsly.Token{whereKeywordMatcher, joinToken, selectorMatcher}

	var targetData []string
	lastMatched := parsly.Invalid
	var matched *parsly.TokenMatch

	for {
		prevPos := cursor.Pos

		switch lastMatched {
		case whereKeyword:
			stmt.Target = buildTarget(targetData, cursor)
			return lastMatched, nil

		case selectorTokenCode:
			targetData = append(targetData, matched.Text(cursor))
			if len(targetData) >= 2 {
				matchable = matchable[:len(matchable)-1]
			}

		case joinTokenCode:
			join := query.NewJoin(matched.Text(cursor))
			if _, err := parseJoin(cursor, join); err != nil {
				return lastMatched, err
			}

			stmt.Joins = append(stmt.Joins, join)
			matchable = matchable[:len(matchable)-1]
		case parsly.EOF:
			stmt.Target = buildTarget(targetData, cursor)
			return lastMatched, nil
		default:
			if matched != nil {
				return lastMatched, cursor.NewError(matchable...)
			}
		}

		if prevPos == cursor.Pos {
			matched = cursor.MatchAfterOptional(whitespaceMatcher, matchable...)
		}

		lastMatched = matched.Code
	}
}

func buildTarget(targetData []string, cursor *parsly.Cursor) del.Target {
	alias := ""
	if len(targetData) >= 2 {
		alias = targetData[1]
	}

	target := del.Target{
		X:        expr.NewSelector(targetData[0]),
		Comments: matchComment(cursor),
		Alias:    alias,
	}
	return target
}

func tryParseDeleteItems(cursor *parsly.Cursor) ([]*del.Item, error) {
	var items []*del.Item
	matchable := []*parsly.Token{fromKeywordMatcher, selectorMatcher}
	for cursor.Pos < cursor.InputSize {
		matched := cursor.MatchAfterOptional(whitespaceMatcher, matchable...)
		switch matched.Code {
		case fromKeyword:
			return items, nil
		case selectorTokenCode:
			selName := matched.Text(cursor)
			comment := matchComment(cursor)
			items = append(items, &del.Item{
				Comments: comment,
				Raw:      selName,
			})

			matched = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
			if matched.Code != nextCode {
				matchable = matchable[:1]
			}

		default:
			return nil, cursor.NewError(matchable...)
		}
	}
	return items, nil
}

func tryParseQualify(cursor *parsly.Cursor, lastMatchedCode int) (*expr.Qualify, error) {
	if lastMatchedCode != whereKeyword {
		return nil, nil
	}

	qualify := expr.NewQualify()
	return qualify, ParseQualify(cursor, qualify)
}

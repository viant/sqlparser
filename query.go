package sqlparser

import (
	"bytes"
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
	"strings"
)

// ParseQuery parses query
func ParseQuery(SQL string) (*query.Select, error) {
	result := &query.Select{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)

	err := parseQuery(cursor, result)
	if err != nil {
		return result, fmt.Errorf("%w, %s, ", err, SQL)
	}
	return result, err
}

func removeSQLComments(SQL string) string {
	lines := strings.Split(SQL, "\n")
	buffer := new(bytes.Buffer)
	for i, line := range lines {
		if i > 0 {
			buffer.WriteString("\n")
		}
		i++
		if index := strings.Index(line, "--"); index != -1 {
			line = line[:index]
		}
		buffer.WriteString(line)
	}
	return buffer.String()
}

func parseQuery(cursor *parsly.Cursor, dest *query.Select) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, withKeywordMatcher, selectKeywordMatcher)

beginMatch:
	switch match.Code {
	case withKeyword:
		if len(dest.WithSelects) > 0 {
			return cursor.NewError(asKeywordMatcher, selectorMatcher)
		}
	With:
		withSelect := &query.WithSelect{X: &query.Select{}}
		dest.WithSelects = append(dest.WithSelects, withSelect)
		match = cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
		if match.Code != identifierCode {
			return cursor.NewError(identifierMatcher)
		}
		withSelect.Alias = match.Text(cursor)
		pos := cursor.Pos
		match = cursor.MatchAfterOptional(whitespaceMatcher, asKeywordMatcher, parenthesesMatcher)
		if match.Code == asKeyword {
			pos = cursor.Pos
			match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
		}
		if match.Code != parenthesesCode {
			return cursor.NewError(asKeywordMatcher, parenthesesMatcher)
		}
		withSelect.Raw = match.Text(cursor)
		SQL := withSelect.Raw[1 : len(withSelect.Raw)-1]
		subCursor := parsly.NewCursor(cursor.Path, []byte(SQL), pos)
		if err := parseQuery(subCursor, withSelect.X); err != nil {
			return err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
		if match.Code == nextCode {
			goto With
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, withKeywordMatcher, selectKeywordMatcher)
		goto beginMatch
	case selectKeyword:
		match = cursor.MatchAfterOptional(whitespaceMatcher, selectionKindMatcher)
		if match.Code == selectionKind {
			dest.Kind = match.Text(cursor)
		}
		dest.List = make(query.List, 0)
		if err := parseSelectListItem(cursor, &dest.List); err != nil {
			return err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, fromKeywordMatcher)
		pos := cursor.Pos
		switch match.Code {
		case fromKeyword:
			dest.From = query.From{}
			match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher, parenthesesMatcher)
			switch match.Code {
			case selectorTokenCode:
				identityOrAlias := match.Text(cursor)
				withSelect := dest.WithSelects.Select(identityOrAlias)
				if withSelect != nil {
					dest.From.X = expr.NewParenthesis(withSelect.Raw)
					dest.From.Alias = identityOrAlias
				} else {
					dest.From.X = expr.NewSelector(identityOrAlias)
				}
			case parenthesesCode:
				dest.From.X = expr.NewRaw(match.Text(cursor))
				rawNode := expr.NewRaw(match.Text(cursor))
				dest.From.X = rawNode
				rawExpr := trimEnclosure(rawNode.Raw)
				rawParser := parsly.NewCursor(cursor.Path, []byte(rawExpr), pos)
				subSelect := &query.Select{}
				if err := parseQuery(rawParser, subSelect); err != nil {
					return fmt.Errorf("invalid subquery: %w, %s", err, rawExpr)
				}
				rawNode.X = subSelect
			}
			if dest.From.Alias == "" {
				dest.From.Alias = discoverAlias(cursor)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				dest.From.Comments = match.Text(cursor)
			}

			dest.Joins = make([]*query.Join, 0)

			match = cursor.MatchAfterOptional(whitespaceMatcher, joinMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
			if match.Code == parsly.EOF {
				return nil
			}
			hasMatch, err := matchPostFrom(cursor, dest, match)
			if !hasMatch && err == nil {
				err = cursor.NewError(joinMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func matchPostFrom(cursor *parsly.Cursor, dest *query.Select, match *parsly.TokenMatch) (bool, error) {

	switch match.Code {
	case joinToken:
		if err := appendJoin(cursor, match, dest); err != nil {
			return false, err
		}
	case whereKeyword:
		dest.Qualify = expr.NewQualify()
		if err := ParseQualify(cursor, dest.Qualify); err != nil {
			return false, err
		}

		match = cursor.MatchAfterOptional(whitespaceMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
		return matchPostFrom(cursor, dest, match)

	case groupByKeyword:
		if err := parseGroupByList(cursor, &dest.GroupBy); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
		return matchPostFrom(cursor, dest, match)

	case havingKeyword:
		dest.Having = expr.NewQualify()
		if err := ParseQualify(cursor, dest.Having); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher)
		return matchPostFrom(cursor, dest, match)
	case orderByKeyword:
		if err := parseOrderByListItem(cursor, &dest.OrderBy); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, windowMatcher, unionMatcher)
		return matchPostFrom(cursor, dest, match)
	case unionKeyword:
		matchedText := match.Text(cursor)
		union := &query.Union{X: &query.Select{}, Raw: matchedText}
		if strings.Contains(strings.ToLower(matchedText), "all") {
			union.Kind = "all"
		}
		dest.Union = union
		err := parseQuery(cursor, union.X)
		return err == nil, err
	case windowTokenCode:
		matchedText := match.Text(cursor)
		dest.Window = expr.NewRaw(matchedText)
		match = cursor.MatchAfterOptional(whitespaceMatcher, intLiteralMatcher)
		if match.Code == intLiteral {
			literal := expr.NewNumericLiteral(match.Text(cursor))
			switch strings.ToLower(matchedText) {
			case "limit":
				dest.Limit = literal
			case "offset":
				dest.Offset = literal
			}
		}
	case parsly.EOF:
		return true, nil
	default:
		return false, nil
	}
	return true, nil
}

func expectExpectIdentifiers(cursor *parsly.Cursor, expect *[]string) (bool, error) {
	match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	switch match.Code {
	case identifierCode:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	default:
		return false, nil
	}

	snapshotPos := cursor.Pos
	match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		has, err := expectExpectIdentifiers(cursor, expect)
		if err != nil {
			return false, err
		}
		if !has {
			cursor.Pos = snapshotPos
			return true, nil
		}
	}
	return true, nil
}

func expectIdentifiers(cursor *parsly.Cursor, expect *[]string) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	switch match.Code {
	case identifierCode:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	default:
		return nil
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		return expectIdentifiers(cursor, expect)
	}
	return nil
}

func expectIdentifiersOrPosition(cursor *parsly.Cursor, expect *[]string) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher, numericLiteralMatcher)
	switch match.Code {
	case identifierCode:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	case numericLiteral:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	default:
		return nil
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		return expectIdentifiers(cursor, expect)
	}
	return nil
}

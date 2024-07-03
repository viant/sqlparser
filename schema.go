package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/schema"
	"strings"
)

// ParseRegisterType parses register type
func ParseRegisterType(SQL string) (*schema.Register, error) {
	result := &schema.Register{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseRegisterType(cursor, result)
	return result, err
}

func parseRegisterType(cursor *parsly.Cursor, destination *schema.Register) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, registerType)
	if match.Code != registerTypeKeyword {
		return cursor.NewError(registerType)
	}
	if match = cursor.MatchAfterOptional(whitespaceMatcher, globalMatcher); match.Code == globalKeyword {
		destination.Global = true
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return cursor.NewError(selectorMatcher)
	}
	destination.Name = match.Text(cursor)
	match = cursor.MatchAfterOptional(whitespaceMatcher, asKeywordMatcher)
	if match.Code != asKeyword {
		return cursor.NewError(asKeywordMatcher)
	}
	destination.Spec = strings.TrimSpace(string(cursor.Input[cursor.Pos:]))
	return nil
}

// ParseRegisterSet parses register set
func ParseRegisterSet(SQL string) (*schema.Register, error) {
	result := &schema.Register{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseRegisterSet(cursor, result)
	return result, err
}

// TODO create parse register object
func parseRegisterSet(cursor *parsly.Cursor, destination *schema.Register) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, registerSet)
	if match.Code != registerSetKeyword {
		return cursor.NewError(registerSet)
	}
	if match = cursor.MatchAfterOptional(whitespaceMatcher, globalMatcher); match.Code == globalKeyword {
		destination.Global = true
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
	if match.Code != selectorTokenCode {
		return cursor.NewError(selectorMatcher)
	}
	destination.Name = match.Text(cursor)
	match = cursor.MatchAfterOptional(whitespaceMatcher, asKeywordMatcher)
	if match.Code != asKeyword {
		return cursor.NewError(asKeywordMatcher)
	}
	destination.Spec = strings.TrimSpace(string(cursor.Input[cursor.Pos:]))
	return nil
}

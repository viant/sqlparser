package sqlparser

import (
	"fmt"
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
	match := cursor.MatchAfterOptional(whitespaceMatcher, registerKeywordMatcher)
	if match.Code != registerKeyword {
		return cursor.NewError(registerKeywordMatcher)
	}
	if match = cursor.MatchAfterOptional(whitespaceMatcher, globalMatcher); match.Code == globalKeyword {
		destination.Global = true
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, typeKeywordMatcher)
	if match.Code != typeKeyword {
		return cursor.NewError(typeKeywordMatcher)
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
func ParseRegisterSet(SQL string) (*schema.RegisterSet, error) {
	result := &schema.RegisterSet{}
	SQL = removeSQLComments(SQL)
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	err := parseRegisterSet(cursor, result)
	return result, err
}

func parseRegisterSet(cursor *parsly.Cursor, destination *schema.RegisterSet) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, registerKeywordMatcher)
	if match.Code != registerKeyword {
		return cursor.NewError(registerKeywordMatcher)
	}
	if match = cursor.MatchAfterOptional(whitespaceMatcher, globalMatcher); match.Code == globalKeyword {
		destination.Global = true
	}
	match = cursor.MatchAfterOptional(whitespaceMatcher, setKeywordMatcher)
	if match.Code != setKeyword {
		return cursor.NewError(setKeywordMatcher)
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, ttlKeywordMatcher)
	if match.Code == ttlKeyword {
		match = cursor.MatchAfterOptional(whitespaceMatcher, intLiteralMatcher)
		if match.Code != intLiteralMatcher.Code {
			return cursor.NewError(setKeywordMatcher)
		}

		ttl64, err := match.Int(cursor)
		if err != nil {
			return fmt.Errorf("parseregisterset unable to get int value due to: %w", err)
		}
		destination.TTL = int(ttl64)
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

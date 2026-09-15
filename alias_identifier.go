package sqlparser

import (
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/matcher"
)

// aliasIdentifier preserves delimited SQL spelling while sharing identifier
// decoding with the native identifier parser.
type aliasIdentifier struct{}

func (aliasIdentifier) Match(cursor *parsly.Cursor) int {
	if cursor.Pos >= len(cursor.Input) {
		return 0
	}
	// Single quotes remain string literals in alias syntax.
	if cursor.Input[cursor.Pos] == '\'' {
		return 0
	}
	parser := tableIdentifierParser{source: string(cursor.Input), position: cursor.Pos}
	if _, err := parser.part(); err != nil {
		return 0
	}
	return parser.position - cursor.Pos
}

// discoverAlias owns optional alias syntax. Once AS or an identifier start is
// present, a failed match is malformed syntax rather than an absent alias.
// Other tokens remain available to the expression/clause owner and its hooks.
func discoverAlias(cursor *parsly.Cursor) (string, error) {
	pos := cursor.Pos
	identifier := aliasIdentifier{}
	explicit := false
	for {
		match := cursor.MatchAfterOptional(whitespaceMatcher, exceptKeywordMatcher, asKeywordMatcher, onKeywordMatcher, fromKeywordMatcher, joinMatcher, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher, unionMatcher, aliasIdentifierMatcher)
		// Clause matchers can match a keyword prefix (for example FROM in
		// from_records). Only a whole token can terminate alias discovery.
		if match.Size > 0 && match.Code != identifierCode && cursor.Pos < len(cursor.Input) {
			start := *cursor
			start.Pos = match.Offset
			if identifier.Match(&start) > match.Size {
				cursor.Pos = match.Offset
				match = cursor.MatchOne(aliasIdentifierMatcher)
			}
		}
		switch match.Code {
		case asKeyword:
			if explicit {
				return "", cursor.NewError(aliasIdentifierMatcher)
			}
			explicit = true
			continue
		case identifierCode:
			alias := match.Text(cursor)
			if !identifier.boundary(cursor) {
				return "", cursor.NewError(aliasIdentifierMatcher)
			}
			return alias, nil
		case exceptKeyword, fromKeyword, onKeyword, orderByKeyword, joinToken, whereKeyword, groupByKeyword, havingKeyword, windowTokenCode, unionKeyword:
			cursor.Pos = match.Offset
		default:
			if cursor.Pos < len(cursor.Input) {
				ch := cursor.Input[cursor.Pos]
				if matcher.IsLetter(ch) || ch >= '0' && ch <= '9' || strings.ContainsRune("\"`[_", rune(ch)) {
					return "", cursor.NewError(aliasIdentifierMatcher)
				}
			}
		}
		if explicit {
			return "", cursor.NewError(aliasIdentifierMatcher)
		}
		cursor.Pos = pos
		return "", nil
	}
}

// boundary prevents a valid alias prefix from hiding a malformed suffix. This
// checks only the alias token, not consumption of the surrounding SQL/template.
func (aliasIdentifier) boundary(cursor *parsly.Cursor) bool {
	if cursor.Pos == len(cursor.Input) {
		return true
	}
	switch cursor.Input[cursor.Pos] {
	case ' ', '\t', '\n', '\r', '\v', '\f', ',', ')', ';':
		return true
	}
	remaining := cursor.Input[cursor.Pos:]
	return len(remaining) >= 2 && (remaining[0] == '/' && remaining[1] == '*' || remaining[0] == '-' && remaining[1] == '-')
}

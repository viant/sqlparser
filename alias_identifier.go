package sqlparser

import (
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
	switch cursor.Input[cursor.Pos] {
	case '"', '`', '[':
		parser := tableIdentifierParser{source: string(cursor.Input), position: cursor.Pos}
		if _, err := parser.part(); err != nil {
			return 0
		}
		return parser.position - cursor.Pos
	default:
		return matcher.NewIdentifier().Match(cursor)
	}
}

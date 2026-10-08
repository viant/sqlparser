package sqlparser

import (
	"unicode"
	"unicode/utf8"

	"github.com/viant/parsly"
)

// MatchBytes is the allocation-free replacement for matchStringDeprecated
// and the implementation selected by Match for normal parser operation.
// It recognizes the same token without copying the whole cursor input or
// decoding a quoted name which the matcher would immediately discard.
// Positions and the returned length remain byte offsets, including for Unicode.
func (aliasIdentifier) MatchBytes(cursor *parsly.Cursor) int {
	input, start := cursor.Input, cursor.Pos
	if start >= len(input) || input[start] == '\'' {
		return 0
	}
	position := start
	quote := input[position]
	if quote == '"' || quote == '`' || quote == '[' {
		close := quote
		if close == '[' {
			close = ']'
		}
		position++
		for position < len(input) {
			current := input[position]
			position++
			if current == 0 {
				return 0
			}
			if current == close {
				if position < len(input) && input[position] == close {
					position++
					continue
				}
				return position - start
			}
		}
		return 0
	}
	for position < len(input) {
		r, size := utf8.DecodeRune(input[position:])
		if r == utf8.RuneError && size == 1 {
			return 0
		}
		allowed := unicode.IsLetter(r) || r == '_' || position > start && (unicode.IsDigit(r) || r == '$')
		if !allowed {
			break
		}
		position += size
	}
	return position - start
}

package sqlparser

import (
	"strings"

	"github.com/viant/parsly"
)

// keywordSequence matches whole words separated by SQL whitespace or comments.
// It does not consume prefixes such as NOTLIKE or LIKE_pattern.
type keywordSequence []string

func (words keywordSequence) Match(cursor *parsly.Cursor) int {
	lookahead := *cursor
	for i, word := range words {
		if i > 0 {
			start := lookahead.Pos
			skipExpressionSpace(&lookahead)
			if lookahead.Pos == start {
				return 0
			}
		}
		size := (aliasIdentifier{}).Match(&lookahead)
		if size != len(word) || !strings.EqualFold(string(lookahead.Input[lookahead.Pos:lookahead.Pos+size]), word) {
			return 0
		}
		lookahead.Pos += size
	}
	return lookahead.Pos - cursor.Pos
}

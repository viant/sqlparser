package matcher

import (
	"github.com/viant/parsly"
)

type stringMatcher struct {
	quote byte
}

// Match matches string
func (m *stringMatcher) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	inputSize := len(input)
	pos := cursor.Pos
	if pos >= inputSize {
		return 0
	}
	value := input[pos]
	if value != m.quote {
		return 0
	}

	matched++
	for i := pos + matched; i < inputSize; i++ {
		value = input[i]
		matched++
		switch value {
		case '\\':
			if i+1 < inputSize {
				i++
				matched++
			}
		case m.quote: //quotes
			if i+1 < inputSize && input[i+1] == m.quote {
				i++
				matched++
				continue
			}
			return matched
		}
	}
	return 0
}

// NewStringMatcher returns a string matcher
func NewStringMatcher(quote byte) *stringMatcher {
	return &stringMatcher{quote: quote}
}

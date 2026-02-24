package matcher

import "github.com/viant/parsly"

type prefixedQuote struct {
	prefixes []byte
	quote    byte
	escape   byte
}

// Match matches quoted strings prefixed by a single-byte marker (for example r'...').
func (m *prefixedQuote) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	inputSize := len(input)
	if pos+1 >= inputSize {
		return 0
	}
	if !hasBytePrefix(m.prefixes, input[pos]) || input[pos+1] != m.quote {
		return 0
	}

	matched = 2
	for i := pos + matched; i < inputSize; i++ {
		value := input[i]
		matched++
		if value == m.escape {
			if i+1 < inputSize {
				i++
				matched++
			}
			continue
		}
		if value == m.quote {
			return matched
		}
	}

	return 0
}

func hasBytePrefix(prefixes []byte, value byte) bool {
	for _, candidate := range prefixes {
		if value == candidate {
			return true
		}
	}
	return false
}

// NewPrefixedQuote returns a matcher for prefixed quote literals.
func NewPrefixedQuote(prefixes []byte, quote, escape byte) parsly.Matcher {
	return &prefixedQuote{
		prefixes: prefixes,
		quote:    quote,
		escape:   escape,
	}
}

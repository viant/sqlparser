package matcher

import (
	"github.com/viant/parsly"
)

type selector struct {
	allowsDashes bool
}

// Match matches a string
func (n *selector) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	size := len(input)
	if startsWithCharacter := IsLetter(input[pos]); startsWithCharacter || input[pos] == '$' {
		pos++
		matched++
	} else if input[pos] == '[' {
		pos++
		matched++
		for i := pos; i < size; i++ {
			pos++
			matched++
			if input[i] == ']' {
				return
			}
		}
		return 0
	} else if input[pos] == '`' {
		pos++
		matched++
		for i := pos; i < size; i++ {
			pos++
			matched++
			if input[i] == '`' {
				return
			}
		}
	} else {
		return 0
	}
	for i := pos; i < size; i++ {
		switch input[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_', '.', ':', '$', '/':
			matched++
			continue
		case '*':
			if i > 0 && input[i-1] == '.' {
				matched++
				return matched
			}
			return matched

		case '-':
			if !n.allowsDashes {
				return matched
			}

			matched++
		default:
			if IsLetter(input[i]) {
				matched++
				continue
			}

			return matched
		}
	}

	return matched
}

// NewSelector creates a selector matcher
func NewSelector(allowDashes bool) parsly.Matcher {
	return &selector{
		allowsDashes: allowDashes,
	}
}

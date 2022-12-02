package matcher

import (
	"github.com/viant/parsly"
)

type placeholder struct{}

//Match matches a string
func (n *placeholder) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	if input[pos] == '?' || input[pos] == ':' || input[pos] == '$' {
		pos++
		matched++
	} else {
		return 0
	}
	size := len(input)
	for i := pos; i < size; i++ {
		switch input[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '_', '.', ':', ']':
			matched++
			continue
		case '*':
			if i > 0 && input[i-1] == '.' {
				matched++
				return matched
			}
			return matched
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

//NewPlaceholder creates a placeholder matcher
func NewPlaceholder() *placeholder {
	return &placeholder{}
}

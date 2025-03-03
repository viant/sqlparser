package matcher

import (
	"github.com/viant/parsly"
)

type selector struct {
	isTable bool
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

	inExpr := false
	for i := pos; i < size; i++ {

		if inExpr {
			matched++
			if input[i] == ']' {
				inExpr = false
			}
			continue
		}

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
			if !n.isTable {
				return matched
			}

			matched++
		case '[':
			if !n.isTable {
				return matched
			}
			matched++
			inExpr = true
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

func NewSelector(allowDashes bool) parsly.Matcher {
	return &selector{
		isTable: allowDashes,
	}
}

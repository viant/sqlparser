package matcher

import (
	"github.com/viant/parsly"
)

type selector struct {
	isTable         bool
	bracedTableOnly bool
}

// Match matches a string
func (n *selector) Match(cursor *parsly.Cursor) (matched int) {
	input := cursor.Input
	pos := cursor.Pos
	size := len(input)
	if pos >= size {
		return 0
	}
	bracedRoot := n.isTable && input[pos] == '$' && pos+1 < size && input[pos+1] == '{'
	if n.bracedTableOnly && !bracedRoot {
		return 0
	}
	if bracedRoot {
		end := pos + 2
		if end >= size || !(IsLetter(input[end]) || input[end] == '_') {
			return 0
		}
		for end < size && (IsLetter(input[end]) || input[end] == '_' || input[end] >= '0' && input[end] <= '9') {
			end++
		}
		if end >= size || input[end] != '}' {
			return 0
		}
		matched, pos = end+1-pos, end+1
		if n.bracedTableOnly && (pos >= size || input[pos] != '.') {
			return 0
		}
		// A braced root is one segment, not an implicit alias or concatenation.
		if pos < size && (IsLetter(input[pos]) || input[pos] == '_' || input[pos] == '$' || input[pos] >= '0' && input[pos] <= '9') {
			return 0
		}
	} else if startsWithCharacter := IsLetter(input[pos]); startsWithCharacter || input[pos] == '$' {
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

		if bracedRoot && input[i] == '.' && (i+1 >= size || input[i+1] == '.' || isWhitespace(input[i+1]) || input[i+1] == ')' || input[i+1] == ',') {
			return 0
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

// NewBracedTableSelector recognizes a qualified table rooted in ${name}.
// It distinguishes a table reference from an ordinary standalone placeholder.
func NewBracedTableSelector() parsly.Matcher {
	return &selector{isTable: true, bracedTableOnly: true}
}

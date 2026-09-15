package sqlparser

import "github.com/viant/parsly"

// selectionModifier shares the native selector boundary, so ordinary names
// such as all_records and distinct_column are not consumed as modifiers.
type selectionModifier struct {
	keywords parsly.Matcher
	selector parsly.Matcher
}

func (m selectionModifier) Match(cursor *parsly.Cursor) int {
	matched := m.keywords.Match(cursor)
	if matched != 0 && m.selector.Match(cursor) > matched {
		return 0
	}
	return matched
}

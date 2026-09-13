package source

import "strings"

type Token string

func (t Token) Find(source string, start int) int { return FindCodeToken(source, string(t), start) }
func (t Token) Contains(source string) bool       { return t.Find(source, 0) >= 0 }
func (t Token) ReplaceAll(source, replacement string) string {
	var result strings.Builder
	previous := 0
	for start := t.Find(source, 0); start >= 0; start = t.Find(source, previous) {
		result.WriteString(source[previous:start])
		result.WriteString(replacement)
		previous = start + len(t)
	}
	result.WriteString(source[previous:])
	return result.String()
}

func FindCodeToken(source, token string, start int) int {
	if token == "" {
		return -1
	}
	scanner := NewCodeScanner(source, start)
	for pos, ok := scanner.Next(); ok; pos, ok = scanner.Next() {
		if !strings.HasPrefix(source[pos:], token) {
			continue
		}
		if pos > 0 && (identifier(token[0]) || token[0] == '$') && identifier(source[pos-1]) {
			continue
		}
		end := pos + len(token)
		if identifier(token[len(token)-1]) && end < len(source) && identifier(source[end]) {
			continue
		}
		return pos
	}
	return -1
}

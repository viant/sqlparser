package source

import "strings"

// ReadGroupString returns the delimited source and its exclusive end offset.
func ReadGroupString(source string, start int, open, close byte) (string, int, bool) {
	if start < 0 || start >= len(source) || source[start] != open {
		return "", start, false
	}
	if open == '[' && close == ']' {
		scanner := &CodeScanner{source: source, position: start}
		protected, ok := scanner.protected()
		if !ok || !protected.closed {
			return "", start, false
		}
		return source[start:protected.end], protected.end, true
	}
	scanner := &CodeScanner{source: source, position: start + 1}
	depth := 1
	for scanner.position < len(source) {
		pos := scanner.position
		if source[pos] == open {
			depth++
			scanner.position++
			continue
		}
		if source[pos] == close {
			depth--
			if depth == 0 {
				return source[start : pos+1], pos + 1, true
			}
			scanner.position++
			continue
		}
		if protected, ok := scanner.protected(); ok {
			scanner.position = protected.end
		} else {
			scanner.position++
		}
	}
	return "", start, false
}

func SplitTopLevelCSV(source string) []string {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	var result []string
	start, depth := 0, 0
	scanner := NewCodeScanner(source, 0)
	for pos, ok := scanner.Next(); ok; pos, ok = scanner.Next() {
		switch source[pos] {
		case '(', '{':
			depth++
		case ')', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				result = append(result, strings.TrimSpace(source[start:pos]))
				start = pos + 1
			}
		}
	}
	return append(result, strings.TrimSpace(source[start:]))
}
func SplitArgs(source string) []string { return SplitTopLevelCSV(source) }
func TrimQuote(source string) string {
	source = strings.TrimSpace(source)
	if len(source) < 2 {
		return source
	}
	first, last := source[0], source[len(source)-1]
	if first == last && (first == '\'' || first == '"' || first == '`') {
		return source[1 : len(source)-1]
	}
	if first == '[' && last == ']' {
		return source[1 : len(source)-1]
	}
	return source
}
func TrimQuotedArgs(args []string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = TrimQuote(arg)
	}
	return result
}

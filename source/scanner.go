// Package source provides source-preserving SQL lexical operations. It owns
// protected regions and nesting; consumers retain their own macro semantics.
package source

import "strings"

type region struct {
	start, end int
	kind       string
	closed     bool
}

type CodeScanner struct {
	source   string
	position int
}

func NewCodeScanner(source string, start int) *CodeScanner {
	if start < 0 {
		start = 0
	}
	scanner := &CodeScanner{source: source}
	// Scan from the beginning so an offset inside a quote cannot expose its data.
	for scanner.position < start && scanner.position < len(source) {
		if protected, ok := scanner.protected(); ok {
			scanner.position = protected.end
		} else {
			scanner.position++
		}
	}
	return scanner
}

func (s *CodeScanner) Next() (int, bool) {
	for s.position < len(s.source) {
		if protected, ok := s.protected(); ok {
			s.position = protected.end
			continue
		}
		position := s.position
		s.position++
		return position, true
	}
	return 0, false
}

func (s *CodeScanner) protected() (region, bool) {
	i := s.position
	text := s.source
	if i >= len(text) {
		return region{}, false
	}
	result := region{start: i, end: len(text)}
	if strings.HasPrefix(text[i:], "--") {
		result.kind = "line comment"
		result.closed = true
		if end := strings.IndexByte(text[i:], '\n'); end >= 0 {
			result.end = i + end
		}
		return result, true
	}
	if strings.HasPrefix(text[i:], "/*") {
		result.kind = "block comment"
		depth := 1
		for pos := i + 2; pos < len(text)-1; pos++ {
			if text[pos:pos+2] == "/*" {
				depth++
				pos++
			} else if text[pos:pos+2] == "*/" {
				depth--
				pos++
				if depth == 0 {
					result.end = pos + 1
					result.closed = true
					break
				}
			}
		}
		return result, true
	}
	quote := text[i]
	if quote == '\'' || quote == '"' || quote == '`' || quote == '[' {
		result.kind = "SQL quoted text"
		if quote == '\'' {
			result.kind = "SQL quoted text"
		}
		if quote == '[' {
			quote = ']'
		}
		for pos := i + 1; pos < len(text); pos++ {
			if text[pos] == '\\' && quote != ']' {
				pos++
				continue
			}
			if text[pos] == quote {
				if pos+1 < len(text) && text[pos+1] == quote {
					pos++
					continue
				}
				result.end = pos + 1
				result.closed = true
				break
			}
		}
		return result, true
	}
	if quote == '$' {
		end := i + 1
		for end < len(text) && identifier(text[end]) {
			end++
		}
		if end < len(text) && text[end] == '$' && (end == i+1 || !digit(text[i+1])) {
			delimiter := text[i : end+1]
			result.kind = "SQL quoted text"
			if close := strings.Index(text[end+1:], delimiter); close >= 0 {
				result.end = end + 1 + close + len(delimiter)
				result.closed = true
			}
			return result, true
		}
	}
	return region{}, false
}

func ProtectionAt(source string, position int) string {
	_, _, kind := ProtectedRangeAt(source, position)
	return kind
}

// ProtectedRangeAt returns the half-open quoted/comment span containing an
// offset. An empty kind means the offset is executable code or out of range.
func ProtectedRangeAt(source string, position int) (int, int, string) {
	if position < 0 || position >= len(source) {
		return 0, 0, ""
	}
	scanner := NewCodeScanner(source, 0)
	for scanner.position <= position {
		if protected, ok := scanner.protected(); ok {
			if position < protected.end {
				return protected.start, protected.end, protected.kind
			}
			scanner.position = protected.end
		} else {
			scanner.position++
		}
	}
	return 0, 0, ""
}

func IsWhitespace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n' || value == '\f' || value == '\v'
}

// MaskLineComments preserves byte offsets and newlines while removing only
// executable SQL line comments. Quoted '--' remains literal content.
func MaskLineComments(text string) string {
	result := []byte(text)
	scanner := NewCodeScanner(text, 0)
	for scanner.position < len(text) {
		if protected, ok := scanner.protected(); ok {
			if protected.kind == "line comment" {
				for i := protected.start; i < protected.end; i++ {
					if result[i] != '\n' && result[i] != '\r' {
						result[i] = ' '
					}
				}
			}
			scanner.position = protected.end
		} else {
			scanner.position++
		}
	}
	return string(result)
}
func digit(value byte) bool { return value >= '0' && value <= '9' }
func identifier(value byte) bool {
	return value == '_' || digit(value) || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= 128
}

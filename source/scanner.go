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
	context  codeContext
	syntax   syntaxContext
}

type codeContext struct {
	position     int
	previousEnd  int
	previousKind string
	syntax       syntaxContext
}

// syntaxContext remembers significant tokens before a bracket. It advances
// lazily, so contextual keyword checks stay linear and skip quoted/commented
// text using the same lexer as the public scanner.
type syntaxContext struct {
	position   int
	last       string
	beforeLast string
}

func (c *syntaxContext) advance(text string, limit int) {
	for c.position < limit {
		start := c.position
		subscript := text[start] == '[' && subscriptStart(text, start, c.beforeLast)
		if protected, ok := protectedAt(text, start, subscript); ok {
			c.position = protected.end
			if protected.kind == "SQL quoted text" {
				c.beforeLast, c.last = c.last, text[start:c.position]
			}
			continue
		}
		c.position++
		if IsWhitespace(text[start]) {
			continue
		}
		if identifier(text[start]) || text[start] == '$' {
			for c.position < len(text) && (identifier(text[c.position]) || text[c.position] == '$') {
				c.position++
			}
		}
		c.beforeLast, c.last = c.last, text[start:c.position]
	}
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

// PreviousSignificant returns the last non-whitespace, non-comment byte before
// the byte just returned by a successful Next. Call it before advancing again.
// Quoted text is significant and has a nonempty kind; -1 means no preceding text.
// A lazy cursor reuses the protected-region lexer and advances through preceding
// SQL once across lookups, for O(n) total time and O(1) space.
func (s *CodeScanner) PreviousSignificant() (position int, kind string) {
	context := &s.context
	scanner := CodeScanner{source: s.source, position: context.position, syntax: context.syntax}
	for scanner.position < s.position-1 {
		if protected, ok := scanner.protected(); ok {
			if protected.kind == "SQL quoted text" {
				context.previousEnd = protected.end
				context.previousKind = protected.kind
			}
			scanner.position = protected.end
		} else {
			if !IsWhitespace(s.source[scanner.position]) {
				context.previousEnd = scanner.position + 1
				context.previousKind = ""
			}
			scanner.position++
		}
	}
	context.position = scanner.position
	context.syntax = scanner.syntax
	return context.previousEnd - 1, context.previousKind
}

func (s *CodeScanner) protected() (region, bool) {
	i := s.position
	text := s.source
	subscript := false
	if i < len(text) && text[i] == '[' {
		s.syntax.advance(text, i)
		subscript = subscriptStart(text, i, s.syntax.beforeLast)
	}
	return protectedAt(text, i, subscript)
}

func protectedAt(text string, i int, subscript bool) (region, bool) {
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
	if quote == '[' && subscript {
		return region{}, false
	}
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
		if size := DollarQuoteDelimiterSize(text[i:]); size > 0 {
			end := i + size
			delimiter := text[i:end]
			result.kind = "SQL quoted text"
			if close := strings.Index(text[end:], delimiter); close >= 0 {
				result.end = end + close + len(delimiter)
				result.closed = true
			}
			return result, true
		}
	}
	return region{}, false
}

// An adjacent bracket after an operand is executable element access. A
// separated bracket, or one after a dot/operator, still quotes an identifier.
func subscriptStart(text string, position int, precedingToken string) bool {
	if position == 0 {
		return false
	}
	previous := text[position-1]
	if !identifier(previous) {
		return strings.ContainsRune(")]}`\"'?$", rune(previous))
	}
	start := position - 1
	for start > 0 && (identifier(text[start-1]) || text[start-1] == '$') {
		start--
	}
	// A qualified name or parameter can use a keyword as its final segment.
	if start > 0 && strings.ContainsRune(".:@", rune(text[start-1])) {
		return true
	}
	// These tokens introduce a name/expression rather than complete an
	// operand. SQL does not require whitespace before a quoted identifier.
	for _, keyword := range []string{
		"SELECT", "AS", "FROM", "JOIN", "WHERE", "ON", "BY", "HAVING",
		"WHEN", "THEN", "ELSE", "CASE", "AND", "OR", "NOT", "IN", "IS",
		"LIKE", "BETWEEN", "DISTINCT", "ALL", "UPDATE", "INTO", "TABLE",
		"USING", "SET", "RETURNING", "WITH", "RECURSIVE", "DELETE", "INSERT",
		"VALUES", "LIMIT", "OFFSET", "ORDER", "GROUP", "UNION", "EXCEPT", "INTERSECT",
	} {
		if strings.EqualFold(text[start:position], keyword) {
			return !bracketNameAfterKeyword(precedingToken, keyword)
		}
	}
	return true
}

func bracketNameAfterKeyword(previous, keyword string) bool {
	switch keyword {
	case "VALUES", "ORDER", "GROUP":
		// VALUES introduces parenthesized rows, and ORDER/GROUP require BY.
		// Before an adjacent bracket these words can be array identifiers.
		return false
	case "BY":
		return strings.EqualFold(previous, "ORDER") || strings.EqualFold(previous, "GROUP") || strings.EqualFold(previous, "PARTITION")
	case "UPDATE", "INTO", "TABLE", "USING", "SET", "RETURNING", "WITH", "RECURSIVE", "DELETE", "INSERT":
		// These statement/clause words can also name values in expressions.
		if len(previous) == 1 && strings.ContainsAny(previous, "([,+-*/%=<>|") {
			return false
		}
		for _, introducer := range []string{"SELECT", "WHERE", "ON", "BY", "HAVING", "WHEN", "THEN", "ELSE", "AND", "OR", "NOT", "IN", "IS", "LIKE", "BETWEEN", "DISTINCT", "ALL", "SET", "RETURNING"} {
			if strings.EqualFold(previous, introducer) {
				return false
			}
		}
	}
	return true
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

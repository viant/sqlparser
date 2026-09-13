package sqlparser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TableIdentifierParts parses one complete qualified table identifier into
// decoded name parts. It does not split dots inside delimited identifiers or
// accept aliases, expressions, placeholders or trailing SQL. Delimiters follow
// SQL doubled-quote rules (including backticks/brackets used by SQL dialects).
// TableName/TableSelector retain their existing AST/string behavior.
func TableIdentifierParts(source string) ([]string, error) {
	if !utf8.ValidString(source) {
		return nil, fmt.Errorf("invalid UTF-8 in table identifier")
	}
	parser := tableIdentifierParser{source: source}
	return parser.parse()
}

type tableIdentifierParser struct {
	source   string
	position int
}

func (p *tableIdentifierParser) spaces() {
	for p.position < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.position:])
		if !unicode.IsSpace(r) {
			return
		}
		p.position += size
	}
}
func (p *tableIdentifierParser) parse() ([]string, error) {
	var parts []string
	for {
		p.spaces()
		if p.position >= len(p.source) {
			return nil, fmt.Errorf("table identifier part is required")
		}
		name, err := p.part()
		if err != nil {
			return nil, err
		}
		if name == "" {
			return nil, fmt.Errorf("empty table identifier part")
		}
		parts = append(parts, name)
		p.spaces()
		if p.position == len(p.source) {
			return parts, nil
		}
		if p.source[p.position] != '.' {
			return nil, fmt.Errorf("unexpected table identifier suffix at %d", p.position)
		}
		p.position++
	}
}
func (p *tableIdentifierParser) part() (string, error) {
	quote := p.source[p.position]
	if quote == '\'' || quote == '"' || quote == '`' || quote == '[' {
		close := quote
		if close == '[' {
			close = ']'
		}
		p.position++
		var result strings.Builder
		for p.position < len(p.source) {
			current := p.source[p.position]
			p.position++
			if current == 0 {
				return "", fmt.Errorf("NUL in table identifier")
			}
			if current == close {
				if p.position < len(p.source) && p.source[p.position] == close {
					result.WriteByte(close)
					p.position++
					continue
				}
				return result.String(), nil
			}
			result.WriteByte(current)
		}
		return "", fmt.Errorf("unterminated table identifier")
	}
	start := p.position
	for p.position < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.position:])
		if r == utf8.RuneError && size == 1 {
			return "", fmt.Errorf("invalid UTF-8 in table identifier")
		}
		allowed := unicode.IsLetter(r) || r == '_' || p.position > start && (unicode.IsDigit(r) || r == '$')
		if !allowed {
			break
		}
		p.position += size
	}
	if p.position == start {
		return "", fmt.Errorf("invalid table identifier at %d", start)
	}
	return p.source[start:p.position], nil
}

package source

import "strings"

func FindTopLevelKeyword(source, keyword string, start int) int {
	depth := 0
	scanner := NewCodeScanner(source, 0)
	words := strings.Fields(keyword)
	if len(words) == 0 {
		return -1
	}
	for pos, ok := scanner.Next(); ok; pos, ok = scanner.Next() {
		switch source[pos] {
		case '(', '{', '[':
			depth++
			continue
		case ')', '}', ']':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth != 0 || pos < start || pos > 0 && identifier(source[pos-1]) {
			continue
		}
		cursor := pos
		matched := true
		for i, word := range words {
			if i > 0 {
				before := cursor
				for cursor < len(source) {
					if IsWhitespace(source[cursor]) {
						cursor++
						continue
					}
					separator := &CodeScanner{source: source, position: cursor}
					if protected, ok := separator.protected(); ok && (protected.kind == "line comment" || protected.kind == "block comment") {
						cursor = protected.end
						continue
					}
					break
				}
				if cursor == before {
					matched = false
					break
				}
			}
			if cursor+len(word) > len(source) || !strings.EqualFold(source[cursor:cursor+len(word)], word) {
				matched = false
				break
			}
			cursor += len(word)
			if cursor < len(source) && identifier(source[cursor]) {
				matched = false
				break
			}
		}
		if matched {
			return pos
		}
	}
	return -1
}
func FindLastTopLevelKeyword(source, keyword string, start int) int {
	last := -1
	for next := FindTopLevelKeyword(source, keyword, start); next >= 0; next = FindTopLevelKeyword(source, keyword, next+len(keyword)) {
		last = next
	}
	return last
}
func HasTopLevelClause(source, clause string) bool {
	return FindTopLevelKeyword(source, clause, 0) >= 0
}

func CriteriaBoundary(source string) int {
	boundary := len(source)
	for _, keyword := range []string{"group by", "having", "order by", "limit", "offset", "union", "intersect", "except", "fetch", "for update"} {
		if pos := FindTopLevelKeyword(source, keyword, 0); pos >= 0 && pos < boundary {
			boundary = pos
		}
	}
	scanner := NewCodeScanner(source, 0)
	depth := 0
	for pos, ok := scanner.Next(); ok; pos, ok = scanner.Next() {
		switch source[pos] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 && pos < boundary {
				boundary = pos
			}
		}
	}
	// Appended predicates must precede trailing comments (especially --), or
	// the database would treat the new condition as comment text.
	scanner = NewCodeScanner(source, 0)
	last := 0
	for scanner.position < len(source) {
		if protected, ok := scanner.protected(); ok {
			if protected.kind == "SQL quoted text" {
				last = protected.end
			}
			scanner.position = protected.end
			continue
		}
		if !IsWhitespace(source[scanner.position]) {
			last = scanner.position + 1
		}
		scanner.position++
	}
	if last < boundary {
		boundary = last
	}
	return boundary
}

func SplitTopLevelAlias(source string) (string, string) {
	if pos := FindLastTopLevelKeyword(source, "as", 0); pos >= 0 {
		return strings.TrimSpace(source[:pos]), strings.TrimSpace(source[pos+2:])
	}
	// SQL permits an unadorned final identifier alias. Restrict inference to
	// whitespace-separated identifiers following a complete source expression.
	scanner := NewCodeScanner(source, 0)
	depth, lastSpace := 0, -1
	for pos, ok := scanner.Next(); ok; pos, ok = scanner.Next() {
		switch source[pos] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && IsWhitespace(source[pos]) {
				lastSpace = pos
			}
		}
	}
	if lastSpace >= 0 {
		core, alias := strings.TrimSpace(source[:lastSpace]), strings.TrimSpace(source[lastSpace:])
		valid := alias != "" && !digit(alias[0])
		for i := 0; i < len(alias); i++ {
			valid = valid && identifier(alias[i])
		}
		if len(alias) > 1 && (alias[0] == '"' || alias[0] == '`' || alias[0] == '[') && TrimQuote(alias) != alias {
			valid = true
		}
		switch strings.ToUpper(alias) {
		case "END", "NULL", "TRUE", "FALSE", "ASC", "DESC":
			valid = false
		}
		if strings.HasSuffix(strings.ToLower(core), " collate") {
			valid = false
		}
		if valid && core != "" && !strings.ContainsAny(core[len(core)-1:], "+-*/=<>|") {
			return core, alias
		}
	}
	return strings.TrimSpace(source), ""
}

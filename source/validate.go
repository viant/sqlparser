package source

import "fmt"

// ValidateStructure checks parentheses, subscripts and protected-region closure without
// imposing a dialect grammar on executable expressions or template semantics.
func ValidateStructure(text string) error {
	scanner := NewCodeScanner(text, 0)
	var groups []int
	for scanner.position < len(text) {
		if region, ok := scanner.protected(); ok {
			if !region.closed {
				return fmt.Errorf("unclosed %s at byte %d", region.kind, region.start)
			}
			scanner.position = region.end
			continue
		}
		pos := scanner.position
		switch text[pos] {
		case '(', '[':
			groups = append(groups, pos)
		case ')', ']':
			if len(groups) == 0 || text[groups[len(groups)-1]] == '(' && text[pos] != ')' || text[groups[len(groups)-1]] == '[' && text[pos] != ']' {
				return fmt.Errorf("unexpected '%c' at byte %d", text[pos], pos)
			}
			groups = groups[:len(groups)-1]
		}
		scanner.position++
	}
	if len(groups) > 0 {
		pos := groups[len(groups)-1]
		return fmt.Errorf("unclosed '%c' at byte %d", text[pos], pos)
	}
	return nil
}

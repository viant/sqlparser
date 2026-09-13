package source

import "fmt"

// ValidateStructure checks parentheses and protected-region closure without
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
		case '(':
			groups = append(groups, pos)
		case ')':
			if len(groups) == 0 {
				return fmt.Errorf("unexpected ')' at byte %d", pos)
			}
			groups = groups[:len(groups)-1]
		}
		scanner.position++
	}
	if len(groups) > 0 {
		return fmt.Errorf("unclosed '(' at byte %d", groups[len(groups)-1])
	}
	return nil
}

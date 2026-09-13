package matcher

import (
	"github.com/viant/parsly"
	"testing"
)

func TestStringMatcher(t *testing.T) {
	for _, tt := range []struct {
		source string
		length int
	}{
		{"", 0}, {"'alice' tail", 7}, {"'O''Brien' tail", 10}, {"'O\\'Brien' tail", 10}, {"'unterminated", 0}, {"''''", 4},
	} {
		t.Run(tt.source, func(t *testing.T) {
			if got := NewStringMatcher('\'').Match(parsly.NewCursor("", []byte(tt.source), 0)); got != tt.length {
				t.Fatalf("length %d expected %d", got, tt.length)
			}
		})
	}
}

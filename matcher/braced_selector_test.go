package matcher

import (
	"github.com/viant/parsly"
	"testing"
)

func TestBracedRootOnlyInTableSelectors(t *testing.T) {
	for _, table := range []bool{false, true} {
		input := []byte("${project}.ds.records rest")
		size := NewSelector(table).Match(parsly.NewCursor("", input, 0))
		if table && string(input[:size]) != "${project}.ds.records" {
			t.Fatalf("table match=%q", input[:size])
		}
		if !table && size > 1 {
			t.Fatalf("value selector grammar broadened: %q", input[:size])
		}
	}
	if size := NewSelector(true).Match(parsly.NewCursor("", nil, 0)); size != 0 {
		t.Fatal(size)
	}
}

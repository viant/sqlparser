package source

import (
	"strings"
	"testing"
)

func TestMaskLineCommentsPreservesOffsetsAndProtectedText(t *testing.T) {
	input := "SELECT '-- literal', $tag$-- dollar$tag$ -- comment\nFROM users -- tail"
	actual := MaskLineComments(input)
	if len(actual) != len(input) {
		t.Fatalf("length = %d, want %d", len(actual), len(input))
	}
	if !strings.Contains(actual, "'-- literal'") || !strings.Contains(actual, "$tag$-- dollar$tag$") {
		t.Fatalf("protected text changed: %q", actual)
	}
	if strings.Contains(actual, "comment") || strings.Contains(actual, "tail") {
		t.Fatalf("line comment was not masked: %q", actual)
	}
	if strings.Index(actual, "FROM") != strings.Index(input, "FROM") {
		t.Fatalf("FROM offset changed: got %d, want %d", strings.Index(actual, "FROM"), strings.Index(input, "FROM"))
	}
}

func TestMaskLineCommentsSupportsEveryLineEnding(t *testing.T) {
	for _, lineEnding := range []string{"\n", "\r\n", "\r"} {
		input := "SELECT 1 -- first" + lineEnding + "SELECT 2 -- second"
		actual := MaskLineComments(input)
		if len(actual) != len(input) {
			t.Fatalf("line ending %q changed length: got %d, want %d", lineEnding, len(actual), len(input))
		}
		if strings.Contains(actual, "first") || strings.Contains(actual, "second") || !strings.Contains(actual, "SELECT 2") {
			t.Fatalf("line ending %q produced %q", lineEnding, actual)
		}
		if actual[len("SELECT 1 -- first"):len("SELECT 1 -- first")+len(lineEnding)] != lineEnding {
			t.Fatalf("line ending %q was not preserved in %q", lineEnding, actual)
		}
	}
}

package main

import (
	"bytes"
	"testing"
)

func TestDeprecatedMatcherOverlay(t *testing.T) {
	source := []byte("func Match() int { " + byteDelegation + " }\nfunc matchStringDeprecated() int { return 1 }")
	original := append([]byte(nil), source...)
	modified, err := deprecatedMatcherOverlay(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("func Match() int { " + stringDelegation + " }\nfunc matchStringDeprecated() int { return 1 }")
	if !bytes.Equal(modified, want) || !bytes.Equal(source, original) {
		t.Fatal("overlay must change only the delegation, without modifying its input")
	}
}

func TestDeprecatedMatcherOverlayRejectsAmbiguousSource(t *testing.T) {
	for _, source := range []string{"no delegation", stringDelegation, byteDelegation + "\n" + byteDelegation} {
		if _, err := deprecatedMatcherOverlay([]byte(source)); err == nil {
			t.Fatal("missing/ambiguous delegation must fail closed")
		}
	}
}

package sqlparser

import "testing"

func TestQueryStructuralValidation(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		fail bool
	}{
		{"SELECT (", true}, {"SELECT f(1", true}, {"SELECT 1)", true},
		{"SELECT 'unclosed", true}, {"SELECT 1 /* unclosed", true}, {"SELECT $$unclosed", true},
		{"SELECT '(' AS text", false}, {"SELECT f((1))", false},
		{"SELECT 1 -- (", false}, {"SELECT 1 /* ( /* ) */ */", false},
		{"SELECT $$($$", false}, {"SELECT [x(y] FROM t", false},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			_, err := ParseQuery(tc.sql, WithStructuralValidation())
			if (err != nil) != tc.fail {
				t.Fatalf("err=%v wantFailure=%v", err, tc.fail)
			}
		})
	}
}

func TestStructuralValidationIsOptIn(t *testing.T) {
	// Legacy callers intentionally keep the existing permissive entrypoint.
	if _, err := ParseQuery("SELECT ("); err != nil {
		t.Fatalf("legacy entrypoint changed: %v", err)
	}
	if _, err := ParseQuery("SELECT (", WithStructuralValidation()); err == nil {
		t.Fatal("strict structure accepted unmatched group")
	}
}

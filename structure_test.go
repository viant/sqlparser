package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
)

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

func TestDollarQuotedProjectionBoundary(t *testing.T) {
	for _, literal := range []string{"$$($$", "$tag$[--)]$tag$"} {
		SQL := "SELECT " + literal + " AS value FROM src WHERE id = 1"
		q, err := ParseQuery(SQL, WithStructuralValidation())
		require.NoError(t, err)
		require.IsType(t, &expr.Literal{}, q.List[0].Expr)
		require.Equal(t, SQL, Stringify(q))
	}
}

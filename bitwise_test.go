package sqlparser

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestBitwisePrecedenceAndAssociativity(t *testing.T) {
	for _, tc := range []struct{ SQL, tree string }{
		{"a | b ^ c & d << e + f * g", "(a | (b ^ (c & (d << (e + (f * g))))))"},
		{"a * b + c << d & e ^ f | g", "((((((a * b) + c) << d) & e) ^ f) | g)"},
		{"a >> b << c", "((a >> b) << c)"},
		{"a << b >> c", "((a << b) >> c)"},
		{"a & b & c", "((a & b) & c)"},
		{"a ^ b ^ c", "((a ^ b) ^ c)"},
		{"a | b | c", "((a | b) | c)"},
		{"a & b = c | d", "((a & b) = (c | d))"},
		{"a << b > c AND d ^ e = f OR g & h <> i", "((((a << b) > c) AND ((d ^ e) = f)) OR ((g & h) <> i))"},
		{"a <= b << c", "(a <= (b << c))"},
		{"a >> b >= c", "((a >> b) >= c)"},
		{"~a & b", "((~ a) & b)"},
		{"a & ~b << c", "(a & ((~ b) << c))"},
		{"~~a", "(~ (~ a))"},
		{"~a[0] & b", "((~ a[0]) & b)"},
		{"(a | b) & c", "(group((a | b)) & c)"},
		{"a << (b >> c)", "(a << group((b >> c)))"},
		{"s.events&(1<<(v.seq))", "(s.events & group((1 << group(v.seq))))"},
		{"128>>2>>1", "((128 >> 2) >> 1)"},
		{"1<<2+3", "(1 << (2 + 3))"},
	} {
		t.Run(tc.SQL, func(t *testing.T) {
			parsed, err := ParseQuery("SELECT " + tc.SQL + " AS result FROM records")
			require.NoError(t, err)
			require.Len(t, parsed.List, 1)
			require.Equal(t, "result", parsed.List[0].Alias)
			require.Equal(t, tc.tree, bitwiseExpressionTree(parsed.List[0].Expr))
			_, direct := (Lineage{Query: parsed}).Compile().Lookup("result")
			require.False(t, direct, "computed values must not become scalar column lineage")
			again, err := ParseQuery(Stringify(parsed))
			require.NoError(t, err)
			require.Equal(t, tc.tree, bitwiseExpressionTree(again.List[0].Expr))
		})
	}
}

func bitwiseExpressionTree(n node.Node) string {
	switch v := n.(type) {
	case *expr.Binary:
		return fmt.Sprintf("(%s %s %s)", bitwiseExpressionTree(v.X), v.Op, bitwiseExpressionTree(v.Y))
	case *expr.Unary:
		return fmt.Sprintf("(%s %s)", v.Op, bitwiseExpressionTree(v.X))
	case *expr.Parenthesis:
		return "group(" + bitwiseExpressionTree(v.X) + ")"
	}
	return Stringify(n)
}

func TestBitwiseForecastingPredicates(t *testing.T) {
	for _, predicate := range []string{
		"BIT_COUNT(v.events & (1 << (v.seq))) > 0",
		"EXISTS(SELECT 1 FROM segments s WHERE value IN (?, ?) AND v.batch_id = s.batch_id AND BIT_COUNT(s.events & (1 << (v.seq))) > 0)",
		"NOT EXISTS(SELECT 1 FROM segments s WHERE value IN (?, ?, ?) AND v.batch_id = s.batch_id AND BIT_COUNT(s.events & (1 << (v.seq))) > 0)",
	} {
		SQL := "SELECT v.country, v.region, SUM(v.avails) AS avails FROM records v WHERE " + predicate + " GROUP BY v.country, v.region"
		parsed, err := ParseQuery(SQL)
		require.NoError(t, err)
		require.Len(t, parsed.GroupBy, 2)
		require.NotNil(t, parsed.Qualify)
		rendered := Stringify(parsed)
		require.Contains(t, rendered, predicate)
		_, err = ParseQuery(rendered)
		require.NoError(t, err)
	}

	// Captured by the supplied reproducer, with all real runtime filters intact.
	fixture, err := os.ReadFile("testdata/forecast_bitwise.sql")
	require.NoError(t, err)
	parsed, err := ParseQuery(string(fixture))
	require.NoError(t, err)
	require.NotNil(t, parsed.Qualify)
	require.Len(t, parsed.GroupBy, 42)
	_, err = ParseQuery(Stringify(parsed))
	require.NoError(t, err)
}

func TestBitwiseRejectsMissingOperands(t *testing.T) {
	for _, expression := range []string{"a &", "a |", "a ^", "a <<", "a >>", "~", "a & & b", "a <<< b", "a >> > b", "a || b", "a +", "a ="} {
		for _, context := range []string{
			"SELECT BIT_COUNT(%s) FROM records",
			"SELECT %s",
			"SELECT %s FROM records",
			"SELECT %s AS result FROM records",
			"SELECT %s, b FROM records",
			"SELECT a FROM records WHERE %s",
			"SELECT a FROM records WHERE %s GROUP BY a",
			"SELECT a FROM records GROUP BY %s",
			"SELECT a FROM records GROUP BY a HAVING %s",
			"SELECT a FROM records ORDER BY %s",
			"SELECT a FROM records JOIN other ON %s",
		} {
			SQL := fmt.Sprintf(context, expression)
			t.Run(SQL, func(t *testing.T) {
				_, err := ParseQuery(SQL)
				require.Error(t, err)
			})
		}
	}
}

func TestBitwiseTraversalAndCollate(t *testing.T) {
	SQL := "SELECT BIT_COUNT((events COLLATE nocase) & (1 << ?)) AS bits FROM records"
	parsed, err := ParseQuery(SQL)
	require.NoError(t, err)
	placeholders := 0
	Traverse(parsed, func(n node.Node) bool {
		if _, ok := n.(*expr.Placeholder); ok {
			placeholders++
		}
		return true
	})
	require.Equal(t, 1, placeholders)
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "(events) & (1 << ?)")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

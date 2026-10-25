package sqlparser

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModuloUsesMultiplicativePrecedence(t *testing.T) {
	for _, tc := range []struct{ SQL, op, left, right string }{
		{"SELECT weekday % 7 + 1 FROM days", "+", "weekday % 7", "1"},
		{"SELECT a % b * c FROM days", "*", "a % b", "c"},
		{"SELECT a * b % c FROM days", "%", "a * b", "c"},
		{"SELECT a % b % c FROM days", "%", "a % b", "c"},
		{"SELECT a+b%7 FROM days", "+", "a", "b % 7"},
	} {
		t.Run(tc.SQL, func(t *testing.T) {
			q, err := ParseQuery(tc.SQL)
			require.NoError(t, err)
			root := requireBinaryOp(t, q.List[0].Expr, tc.op)
			require.Equal(t, tc.left, Stringify(root.X))
			require.Equal(t, tc.right, Stringify(root.Y))
			_, err = ParseQuery(Stringify(q))
			require.NoError(t, err)
		})
	}
}
func TestModuloDaypartDerivedSource(t *testing.T) {
	const SQL = "SELECT * FROM (SELECT ((second_day.weekday % 7) + 1) AS WEEKDAY FROM days second_day) pacing"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	require.Contains(t, Stringify(q), "second_day.weekday % 7")
	require.Equal(t, "pacing", q.From.Alias)
	for _, SQL := range []string{"SELECT a % FROM days", "SELECT % 7 FROM days", "SELECT a % % 7 FROM days"} {
		_, err = ParseQuery(SQL)
		require.Error(t, err)
	}
}

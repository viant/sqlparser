package sqlparser

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"testing"
)

func TestUnarySignOperands(t *testing.T) {
	for _, value := range []string{"-(CASE WHEN active = 1 THEN 60 ELSE 0 END)", "-amount", "+amount", "-COALESCE(amount,0)", "-(amount + 1)"} {
		q, err := ParseQuery("SELECT "+value+" * 2 AS result FROM records", WithStructuralValidation())
		require.NoError(t, err, value)
		product, ok := q.List[0].Expr.(*expr.Binary)
		require.True(t, ok, value)
		require.Equal(t, "*", product.Op)
		_, ok = product.X.(*expr.Unary)
		require.True(t, ok, value)
		_, err = ParseQuery(Stringify(q), WithStructuralValidation())
		require.NoError(t, err)
	}
	for _, value := range []string{"-", "+", "-(", "-()"} {
		_, err := ParseQuery("SELECT "+value+" FROM records", WithStructuralValidation())
		require.Error(t, err, value)
	}
	q, err := ParseQuery("SELECT -1, +2, 5-2 FROM records")
	require.NoError(t, err)
	_, ok := q.List[0].Expr.(*expr.Literal)
	require.True(t, ok, "signed literals keep existing representation")
}

func TestConcatenationOperator(t *testing.T) {
	const SQL = "SELECT CAST(-(CASE WHEN active = 1 THEN 60 ELSE 0 END) AS TEXT) || ' minutes' AS delta FROM records"
	q, err := ParseQuery(SQL, WithStructuralValidation())
	require.NoError(t, err)
	require.Equal(t, "||", q.List[0].Expr.(*expr.Binary).Op)
	_, err = ParseQuery(Stringify(q), WithStructuralValidation())
	require.NoError(t, err)
	q, err = ParseQuery("SELECT a || b || c, 1 | 2, 1 + a || b FROM records")
	require.NoError(t, err)
	concat := q.List[0].Expr.(*expr.Binary)
	require.Equal(t, "||", concat.Op)
	require.Equal(t, "||", concat.X.(*expr.Binary).Op)
	require.Equal(t, "|", q.List[1].Expr.(*expr.Binary).Op)
	require.Equal(t, "+", q.List[2].Expr.(*expr.Binary).Op)
	for _, value := range []string{"a ||", "|| b", "a ||| b"} {
		_, err = ParseQuery("SELECT "+value+" FROM records", WithStructuralValidation())
		require.Error(t, err, value)
	}
}

package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func TestFunctionCallCommentSeparators(t *testing.T) {
	for _, expression := range []string{
		"STRUCT/*comment*/(1 AS id)",
		"STRUCT /*comment*/ (1 AS id)",
		"STRUCT--comment\n(1 AS id)",
		"TO_JSON_STRING/*json*/(STRUCT/*field*/(? AS id))",
		"COALESCE/*first*//*second*/(?, 0)",
		"CAST/*type*/(NULL AS STRING)",
		"OFFSET/*index*/(0)",
		"ARRAY/*query*/(SELECT AS STRUCT id FROM orders)",
	} {
		t.Run(expression, func(t *testing.T) {
			call, err := ParseCallExpr(expression)
			require.NoError(t, err)
			require.NotEmpty(t, call.Args)
			parsed, err := ParseQuery("SELECT " + expression + " AS value FROM orders")
			require.NoError(t, err)
			require.Equal(t, "value", parsed.List[0].Alias)
			require.IsType(t, &expr.Call{}, parsed.List[0].Expr)
			SQL := Stringify(parsed)
			again, err := ParseQuery(SQL)
			require.NoError(t, err)
			require.Equal(t, SQL, Stringify(again))
		})
	}
}

func TestCallLookaheadPreservesNonCallComments(t *testing.T) {
	for _, SQL := range []string{
		"SELECT id /*field*/ AS alias FROM orders",
		"SELECT id/*field*/ alias FROM orders",
		"SELECT id /*field*/ FROM orders",
		"SELECT o.* /*star*/ FROM orders o",
		"SELECT id [alias] FROM orders",
	} {
		parsed, err := ParseQuery(SQL)
		require.NoError(t, err)
		require.Len(t, parsed.List, 1)
		_, isCall := parsed.List[0].Expr.(*expr.Call)
		require.False(t, isCall)
		rendered := Stringify(parsed)
		again, err := ParseQuery(rendered)
		require.NoError(t, err)
		require.Equal(t, rendered, Stringify(again))
	}
}

func TestCommentedStructCallTransformation(t *testing.T) {
	SQL := "SELECT TO_JSON_STRING/*json*/(STRUCT/*fields*/(name COLLATE nocase AS label)) AS result FROM orders"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	parsed, err := ParseQuery(stripped)
	require.NoError(t, err)
	field := parsed.List[0].Expr.(*expr.Call).Args[0].(*expr.Call).Args[0].(*query.Item)
	require.Equal(t, "label", field.Alias)
	require.Equal(t, "name", Stringify(field.Expr))
}

package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestNamedArgumentFieldAccess(t *testing.T) {
	const expression = "AI.EMBED(?, endpoint => 'text-embedding-005', task_type => 'RETRIEVAL_QUERY').result"
	q, err := ParseQuery("SELECT " + expression + " AS embedding FROM documents")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		field, ok := q.List[0].Expr.(*expr.FieldAccess)
		require.True(t, ok)
		require.Equal(t, "result", field.Name)
		call, ok := field.X.(*expr.Call)
		require.True(t, ok)
		require.Len(t, call.Args, 3)
		require.IsType(t, &expr.Placeholder{}, call.Args[0])
		for j, name := range []string{"endpoint", "task_type"} {
			arg, ok := call.Args[j+1].(*expr.Binary)
			require.True(t, ok)
			require.Equal(t, "=>", arg.Op)
			require.Equal(t, &expr.Raw{Raw: name}, arg.X)
			require.Equal(t, &expr.Literal{Kind: "string", Value: []string{"'text-embedding-005'", "'RETRIEVAL_QUERY'"}[j]}, arg.Y)
		}
		require.Equal(t, expression, Stringify(field))
		q, err = ParseQuery(Stringify(q), WithStructuralValidation())
		require.NoError(t, err)
	}
}

func TestNamedArgumentsRoundTripAndTraversal(t *testing.T) {
	for _, expression := range []string{
		"f(value => ?)",
		"f(value=>?)",
		"f($first, endpoint => $second, task_type => g(value => $third)).result",
		"f(task_type => 'QUERY', endpoint => 'model')",
		"f(`value` /* name */ => /* value */ 1 + 2 * 3)",
		"f(value => STRUCT(1 AS x), fallback => values[SAFE_OFFSET(0)])",
		"f(value => (SELECT MAX(id) FROM documents))",
		"f(x = ?, x >= ?, x <= ?, x > ?)",
	} {
		t.Run(expression, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + expression)
			require.NoError(t, err)
			require.Equal(t, expression, Stringify(q.List[0].Expr))
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, q.List[0].Expr, again.List[0].Expr)
		})
	}
	// Query parsing strips line comments before parsing expressions.
	commented, err := ParseQuery("SELECT f(value -- name\n => -- value\n ?)")
	require.NoError(t, err)
	again, err := ParseQuery(Stringify(commented))
	require.NoError(t, err)
	require.Equal(t, commented.List[0].Expr, again.List[0].Expr)
	SQL := "SELECT f($first, endpoint => ($second COLLATE nocase), task_type => g(value => $third)).result"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	stripped = strings.Join(strings.Fields(stripped), " ")
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "endpoint => ($second)")
	require.Contains(t, stripped, "task_type => g(value => $third)")
	require.Contains(t, stripped, ").result")
	for _, input := range []string{SQL, stripped} {
		q, err := ParseQuery(input)
		require.NoError(t, err)
		var placeholders []string
		Traverse(q, func(n node.Node) bool {
			if p, ok := n.(*expr.Placeholder); ok {
				placeholders = append(placeholders, p.Name)
			}
			if ident, ok := n.(*expr.Ident); ok {
				require.NotContains(t, []string{"endpoint", "task_type", "value"}, ident.Name)
			}
			return true
		})
		require.Equal(t, []string{"$first", "$second", "$third"}, placeholders)
	}
}

func TestNamedArgumentsRejectMalformed(t *testing.T) {
	for _, expression := range []string{
		"f(value =>)", "f(=> 1)", "f(value => 1,)",
		"f(value => 1, 2)", "f(value => 1, value => 2)", "f(value => 1, VALUE => 2)",
		"f(value => 1, `value` => 2)", "f(a.b => 1)", "f('value' => 1)",
		"f(1 => 2)", "f(? => 2)", "f(value = > 1)", "f(value => 1 +)",
		"f(value => 1 => 2)", "f(value => DISTINCT x)", "f(value => x ORDER BY y)",
		"f(x ORDER BY value => 1)", "f((value => 1))", "(value => 1)", "value => 1",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression)
			require.Error(t, err)
		})
	}
}

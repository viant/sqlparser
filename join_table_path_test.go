package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func TestJoinHyphenatedTablePath(t *testing.T) {
	for _, path := range []string{"sample-project.analytics.flights", "`sample-project.analytics.flights`", "`sample-project`.analytics.`flights`"} {
		for _, kind := range []string{"JOIN", "INNER JOIN", "LEFT JOIN"} {
			for _, alias := range []string{"f", "AS f"} {
				t.Run(kind+"/"+path+"/"+alias, func(t *testing.T) {
					from, err := ParseQuery("SELECT f.id FROM " + path + " " + alias)
					require.NoError(t, err)
					SQL := "SELECT c.id FROM campaigns c " + kind + " " + path + " " + alias + " ON f.campaign_id = c.id"
					parsed, err := ParseQuery(SQL)
					require.NoError(t, err)
					check := func(q *query.Select) {
						require.Len(t, q.Joins, 1)
						join := q.Joins[0]
						require.Equal(t, path, Stringify(join.With))
						require.Equal(t, from.From.X, join.With)
						require.Equal(t, "f", join.Alias)
						require.NotNil(t, join.On)
						require.IsType(t, &expr.Binary{}, join.On.X)
						require.Equal(t, "(f.campaign_id = c.id)", bitwiseExpressionTree(join.On.X))
					}
					check(parsed)
					again, err := ParseQuery(Stringify(parsed))
					require.NoError(t, err)
					check(again)
				})
			}
		}
	}
}

func TestJoinTablePathsPreservePredicatesAndSubtraction(t *testing.T) {
	SQL := "SELECT c.total-c.used AS remaining FROM campaigns c " +
		"JOIN sample-project.analytics.flights f ON f.campaign_id=c.id " +
		"LEFT JOIN second-project.analytics.budgets AS b ON b.flight_id=f.id-c.offset " +
		"CROSS JOIN UNNEST(b.amount-b.spent) amount WHERE c.total-c.used>0"
	parsed, err := ParseQuery(SQL)
	require.NoError(t, err)
	checkJoinTablePaths(t, parsed)
	again, err := ParseQuery(Stringify(parsed))
	require.NoError(t, err)
	checkJoinTablePaths(t, again)
}

func checkJoinTablePaths(t *testing.T, q *query.Select) {
	t.Helper()
	require.Equal(t, "(c.total - c.used)", bitwiseExpressionTree(q.List[0].Expr))
	require.Len(t, q.Joins, 3)
	for i, expected := range []struct{ path, alias, predicate string }{
		{"sample-project.analytics.flights", "f", "(f.campaign_id = c.id)"},
		{"second-project.analytics.budgets", "b", "(b.flight_id = (f.id - c.offset))"},
	} {
		join := q.Joins[i]
		require.Equal(t, expected.path, Stringify(join.With))
		require.Equal(t, expected.alias, join.Alias)
		require.NotNil(t, join.On)
		require.Equal(t, expected.predicate, bitwiseExpressionTree(join.On.X))
	}
	call, ok := q.Joins[2].With.(*expr.Call)
	require.True(t, ok)
	require.Equal(t, "UNNEST", Stringify(call.X))
	require.Len(t, call.Args, 1)
	require.Equal(t, "(b.amount - b.spent)", bitwiseExpressionTree(call.Args[0]))
	require.Equal(t, "amount", q.Joins[2].Alias)
	require.Nil(t, q.Joins[2].On)
	require.NotNil(t, q.Qualify)
	require.Equal(t, "((c.total - c.used) > 0)", bitwiseExpressionTree(q.Qualify.X))
}

func TestJoinTablePathPreservesSubscript(t *testing.T) {
	q, err := ParseQuery("SELECT a.id FROM records a CROSS JOIN a.items[SAFE_OFFSET(?)] item")
	require.NoError(t, err)
	require.Len(t, q.Joins, 1)
	subscript, ok := q.Joins[0].With.(*expr.Subscript)
	require.True(t, ok, "join target must retain its subscript AST, got %T", q.Joins[0].With)
	require.Equal(t, "a.items", Stringify(subscript.X))
	index, ok := subscript.Index.(*expr.Call)
	require.True(t, ok)
	require.Equal(t, "SAFE_OFFSET", Stringify(index.X))
	require.Len(t, index.Args, 1)
	require.IsType(t, &expr.Placeholder{}, index.Args[0])
	require.Equal(t, "item", q.Joins[0].Alias)
}

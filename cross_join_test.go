package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestCrossJoinWithoutOn(t *testing.T) {
	for _, keyword := range []string{"CROSS JOIN", "cross join", "CrOsS  JoIn", "CROSS\nJOIN", "CROSS\tJOIN"} {
		for _, suffix := range []string{
			"", " WHERE fi.id = ?", " GROUP BY fi.id HAVING COUNT(*) > 0 ORDER BY fi.id LIMIT 20",
			" JOIN features f ON f.id = fi.id WHERE f.id > ?",
			" UNION ALL SELECT id FROM features",
		} {
			t.Run(keyword+suffix, func(t *testing.T) {
				SQL := "SELECT fi.id FROM UNNEST(ao.hourly_stats) hourly " + keyword +
					" UNNEST(hourly.items) line " + keyword +
					" UNNEST(line.metrics) fi" + suffix
				parsed, err := ParseQuery(SQL)
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(parsed.Joins), 2)
				for i, arg := range []string{"hourly.items", "line.metrics"} {
					join := parsed.Joins[i]
					require.Equal(t, keyword, join.Raw)
					require.Nil(t, join.On)
					require.Equal(t, node.Span{}, join.OnSpan)
					call := join.With.(*expr.Call)
					require.Equal(t, "UNNEST", Stringify(call.X))
					require.Len(t, call.Args, 1)
					require.Equal(t, arg, Stringify(call.Args[0]))
					require.Less(t, join.Span.Begin, join.Span.End)
					require.LessOrEqual(t, int(join.Span.End), len(SQL))
					require.Equal(t, keyword+" UNNEST("+arg+") "+join.Alias, SQL[join.Span.Begin:join.Span.End])
				}
				require.Equal(t, "line", parsed.Joins[0].Alias)
				require.Equal(t, "fi", parsed.Joins[1].Alias)
				if strings.Contains(suffix, " JOIN features") {
					require.Len(t, parsed.Joins, 3)
					require.NotNil(t, parsed.Joins[2].On)
				}
				rendered := (Stringifier{PreserveWindow: true}).String(parsed)
				again, err := ParseQuery(rendered)
				require.NoError(t, err)
				require.Equal(t, rendered, (Stringifier{PreserveWindow: true}).String(again))
			})
		}
	}
	parsed, err := ParseQuery("SELECT * FROM orders CROSS JOIN features")
	require.NoError(t, err)
	require.Len(t, parsed.Joins, 1)
	require.Nil(t, parsed.Joins[0].On)
}

func TestConditionalJoinsStillRequireOn(t *testing.T) {
	for _, keyword := range []string{"JOIN", "INNER JOIN", "LEFT JOIN", "LEFT OUTER JOIN"} {
		for _, prefix := range []string{"SELECT * FROM orders o ", "SELECT * FROM orders o CROSS JOIN features f "} {
			for _, suffix := range []string{"", " WHERE a.id = ?", " CROSS JOIN features f"} {
				_, err := ParseQuery(prefix + keyword + " accounts a" + suffix)
				require.Error(t, err, prefix+keyword+suffix)
			}
		}
	}
}

func TestCorrelatedArrayCrossJoins(t *testing.T) {
	for _, acl := range []string{"ao.id IN (?)", "ao.id IN (?) AND ao.tenant_id = ?"} {
		SQL := "SELECT ao.id, TO_JSON_STRING(ARRAY(SELECT AS STRUCT fi.id, COUNT(*) AS incidents " +
			"FROM UNNEST(ao.hourly_stats) hourly " +
			"CROSS JOIN UNNEST(hourly.items) line " +
			"CROSS JOIN UNNEST(line.metrics) fi " +
			"WHERE fi.id > ? GROUP BY fi.id ORDER BY fi.id LIMIT 20)) AS item_metrics_json " +
			"FROM orders ao WHERE " + acl
		parsed, err := ParseQuery(SQL)
		require.NoError(t, err)
		inner := parsed.List[1].Expr.(*expr.Call).Args[0].(*expr.Call).Args[0].(*query.Select)
		require.Len(t, inner.Joins, 2)
		require.NotNil(t, inner.Qualify)
		require.Len(t, inner.GroupBy, 1)
		require.Len(t, inner.OrderBy, 1)
		require.Equal(t, "20", inner.Limit.Value)
		require.NotNil(t, parsed.Qualify)
		rendered := Stringify(parsed)
		again, err := ParseQuery(rendered)
		require.NoError(t, err)
		require.Equal(t, rendered, Stringify(again))
	}
}

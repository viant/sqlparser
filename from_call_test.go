package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestPrimaryFromFunction(t *testing.T) {
	for _, tc := range []struct {
		SQL, name, alias string
		args             int
	}{
		{"SELECT * FROM UNNEST(ao.hourly_stats) hourly", "UNNEST", "hourly", 1},
		{"SELECT * FROM unnest(ao.hourly_stats) AS hourly", "unnest", "hourly", 1},
		{"SELECT * FROM UNNEST(ao.hourly_stats)", "UNNEST", "", 1},
		{"SELECT * FROM UNNEST /* source */ (ao.hourly_stats) hourly", "UNNEST", "hourly", 1},
		{"SELECT * FROM UNNEST/*source*/(ao.hourly_stats) hourly", "UNNEST", "hourly", 1},
		{"SELECT * FROM UNNEST/*first*//*second*/(ao.hourly_stats) AS hourly", "UNNEST", "hourly", 1},
		{"SELECT * FROM UNNEST-- source\n(ao.hourly_stats) hourly", "UNNEST", "hourly", 1},
		{"SELECT * FROM analytics.hourly_stats/*source*/(ao.hourly_stats) hourly", "analytics.hourly_stats", "hourly", 1},
		{"SELECT * FROM analytics.hourly_stats(?, SAFE_OFFSET(0)) AS hourly", "analytics.hourly_stats", "hourly", 2},
		{"SELECT * FROM hourly_stats() hourly", "hourly_stats", "hourly", 0},
		{"SELECT hourly.hour FROM UNNEST(ao.hourly_stats) hourly JOIN hours h ON h.hour = hourly.hour WHERE h.hour > ?", "UNNEST", "hourly", 1},
	} {
		t.Run(tc.SQL, func(t *testing.T) {
			parsed, err := ParseQuery(tc.SQL)
			require.NoError(t, err)
			call, ok := parsed.From.X.(*expr.Call)
			require.True(t, ok, "source type: %T", parsed.From.X)
			require.Equal(t, tc.name, Stringify(call.X))
			require.Len(t, call.Args, tc.args)
			require.Equal(t, tc.alias, parsed.From.Alias)
			require.Empty(t, TableName(parsed))
			require.Nil(t, TableSelector(parsed))
			table, parenthesized, err := SourceTable(parsed.From.X)
			require.NoError(t, err)
			require.Empty(t, table)
			require.False(t, parenthesized)
			require.Empty(t, (Lineage{Query: parsed}).RootTable())
			_, direct := (Lineage{Query: parsed}).Compile().Lookup("hour")
			require.False(t, direct)
			rendered := Stringify(parsed)
			again, err := ParseQuery(rendered)
			require.NoError(t, err)
			require.Equal(t, rendered, Stringify(again))
			require.IsType(t, &expr.Call{}, again.From.X)
		})
	}
}

func TestCorrelatedArrayWithPrimaryUnnest(t *testing.T) {
	for _, predicate := range []string{"ao.id IN (?)", "ao.id IN (?) AND ao.tenant_id = ?"} {
		t.Run(predicate, func(t *testing.T) {
			SQL := "SELECT ao.id, TO_JSON_STRING(ARRAY(SELECT AS STRUCT hourly.hour, SUM(hourly.bids) AS bids " +
				"FROM UNNEST(ao.hourly_stats) hourly WHERE hourly.hour >= ? " +
				"GROUP BY hourly.hour ORDER BY hourly.hour LIMIT 20)) AS summary_json FROM orders ao WHERE " + predicate
			parsed, err := ParseQuery(SQL)
			require.NoError(t, err)
			jsonCall := parsed.List[1].Expr.(*expr.Call)
			arrayCall := jsonCall.Args[0].(*expr.Call)
			inner := arrayCall.Args[0].(*query.Select)
			from := inner.From.X.(*expr.Call)
			require.Equal(t, "UNNEST", Stringify(from.X))
			require.Equal(t, "ao.hourly_stats", Stringify(from.Args[0]))
			require.Equal(t, "hourly", inner.From.Alias)
			require.Len(t, inner.GroupBy, 1)
			require.Len(t, inner.OrderBy, 1)
			require.NotNil(t, inner.Qualify)
			require.Equal(t, "20", inner.Limit.Value)
			require.NotNil(t, parsed.Qualify)
			found := false
			Traverse(parsed, func(n node.Node) bool {
				if n == from.Args[0] {
					found = true
				}
				return true
			})
			require.True(t, found, "traversal must reach the correlated source argument")
			for _, q := range []*query.Select{parsed, inner} {
				rendered := (Stringifier{PreserveWindow: true}).String(q)
				again, err := ParseQuery(rendered)
				require.NoError(t, err)
				require.Equal(t, rendered, (Stringifier{PreserveWindow: true}).String(again))
			}
		})
	}
}

func TestPrimaryFromFunctionRejectsMalformedArguments(t *testing.T) {
	for _, source := range []string{
		"UNNEST(ao.hourly_stats", "UNNEST(,ao.hourly_stats)",
		"UNNEST(ao.hourly_stats,)", "UNNEST(ao.hourly_stats +)",
		"UNNEST(ao.hourly_stats))", "UNNEST(ao.hourly_stats) h extra",
	} {
		_, err := ParseQuery("SELECT * FROM " + source)
		require.Error(t, err, source)
	}
}

func TestPrimaryFromFunctionStripCollate(t *testing.T) {
	SQL := "SELECT TO_JSON_STRING(ARRAY(SELECT AS STRUCT hourly.hour FROM " +
		"UNNEST(ao.hourly_stats COLLATE nocase) hourly LIMIT 20)) AS summary_json FROM orders ao"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	parsed, err := ParseQuery(stripped)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "UNNEST(ao.hourly_stats) hourly LIMIT 20")
	require.Equal(t, "orders", TableName(parsed))
}

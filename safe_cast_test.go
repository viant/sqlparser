package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestSafeCast(t *testing.T) {
	for _, tc := range []struct{ expression, operand, target string }{
		{"SAFE_CAST(Campaign AS INT64)", "Campaign", "INT64"},
		{"safe_cast(Campaign as INT64)", "Campaign", "INT64"},
		{"SAFE_CAST(COALESCE(?, Campaign) AS DECIMAL(10,2))", "COALESCE(?, Campaign)", "DECIMAL(10,2)"},
		{"SAFE_CAST(CAST(Campaign AS STRING) AS INT64)", "CAST(Campaign AS STRING)", "INT64"},
		{"SAFE_CAST(COALESCE(' AS ', Campaign) /* AS ignored */ AS STRING)", "COALESCE(' AS ', Campaign)", "STRING"},
		{"SAFE_CAST(? AS STRING FORMAT 'BASE64')", "?", "STRING FORMAT 'BASE64'"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + tc.expression + " AS campaign_id FROM records")
			require.NoError(t, err)
			call, ok := q.List[0].Expr.(*expr.Call)
			require.True(t, ok)
			direct, err := ParseCallExpr(tc.expression)
			require.NoError(t, err)
			for _, parsed := range []*expr.Call{call, direct} {
				require.Equal(t, "SAFE_CAST", strings.ToUpper(Stringify(parsed.X)))
				require.Len(t, parsed.Args, 1)
				arg, ok := parsed.Args[0].(*expr.Binary)
				require.True(t, ok)
				require.Equal(t, "AS", strings.ToUpper(arg.Op))
				require.Equal(t, tc.operand, Stringify(arg.X))
				require.Equal(t, &expr.Raw{Raw: tc.target}, arg.Y)
			}
			require.Equal(t, tc.expression, Stringify(call))
			column := NewColumn(q.List[0])
			require.Equal(t, tc.expression, column.Expression)
			require.Equal(t, "campaign_id", column.Alias)
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, call, again.List[0].Expr)
		})
	}
}

func TestSafeCastTraversalAndTransform(t *testing.T) {
	SQL := "SELECT SAFE_CAST(COALESCE($value COLLATE nocase, $fallback) AS INT64) AS campaign_id FROM records WHERE id = $id"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "SAFE_CAST(")
	for _, input := range []string{SQL, stripped} {
		q, err := ParseQuery(input)
		require.NoError(t, err)
		var placeholders []string
		Traverse(q, func(n node.Node) bool {
			if p, ok := n.(*expr.Placeholder); ok {
				placeholders = append(placeholders, p.Name)
			}
			return true
		})
		require.Equal(t, []string{"$value", "$fallback", "$id"}, placeholders)
	}
}

func TestSafeCastRejectsMalformed(t *testing.T) {
	for _, expression := range []string{
		"SAFE_CAST()", "SAFE_CAST(Campaign)", "SAFE_CAST(Campaign, 'INT64')",
		"SAFE_CAST(AS INT64)", "SAFE_CAST(Campaign AS)",
		"SAFE_CAST(Campaign + AS INT64)", "SAFE_CAST(Campaign AS INT64, STRING)",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression + " FROM records")
			require.Error(t, err)
		})
	}
}

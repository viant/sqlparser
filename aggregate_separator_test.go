package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestAggregateSeparator(t *testing.T) {
	for _, expression := range []string{
		"GROUP_CONCAT(DISTINCT channel_v2.channel_name ORDER BY channel_v2.bit_value SEPARATOR ',')",
		"GROUP_CONCAT(name SEPARATOR '')",
		"GROUP_CONCAT(name SEPARATOR \"; \" )",
		"GROUP_CONCAT(first_name, last_name ORDER BY rank DESC, id ASC SEPARATOR ', ')",
		"GROUP_CONCAT(name ORDER BY separator_value SEPARATOR '|')",
		"GROUP_CONCAT(separator, separator_value ORDER BY separator_rank)",
		"GROUP_CONCAT(name ORDER BY rank separator/*comment*/',')",
		"GROUP_CONCAT(name /* before */ SEPARATOR /* value */ ',' /* after */)",
		"GROUP_CONCAT(name ORDER BY rank SEPARATOR ',' LIMIT 20)",
		"GROUP_CONCAT(name SEPARATOR ', LIMIT ORDER BY SEPARATOR')",
		"COALESCE(GROUP_CONCAT(name ORDER BY rank SEPARATOR ','), '')",
	} {
		t.Run(expression, func(t *testing.T) {
			SQL := "SELECT category, " + expression + " AS names FROM records WHERE id > ? GROUP BY category"
			q, err := ParseQuery(SQL, WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, expression, Stringify(q.List[1].Expr))
			require.NotNil(t, q.Qualify)
			require.Len(t, q.GroupBy, 1)
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, Stringify(q), Stringify(again))
			_, err = ParseCallExpr(expression)
			require.NoError(t, err)
		})
	}
}

func TestAggregateSeparatorTreeAndTransforms(t *testing.T) {
	SQL := "SELECT GROUP_CONCAT(DISTINCT name COLLATE nocase ORDER BY rank DESC, id ASC SEPARATOR ',') AS names FROM records"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	call := q.List[0].Expr.(*expr.Call)
	require.Len(t, call.Args, 1)
	separator := call.Args[0].(*expr.Binary)
	require.Equal(t, "SEPARATOR", separator.Op)
	require.Equal(t, "','", separator.Y.(*expr.Literal).Value)
	ordering := separator.X.(*expr.Binary)
	require.Equal(t, "ORDER BY", ordering.Op)
	require.Equal(t, "DISTINCT", ordering.X.(*expr.Unary).Op)
	items := ordering.Y.(query.List)
	require.Len(t, items, 2)
	require.Equal(t, "DESC", items[0].Direction)
	require.Equal(t, "ASC", items[1].Direction)
	visited := false
	Traverse(q, func(n node.Node) bool {
		if n == separator.Y {
			visited = true
		}
		return true
	})
	require.True(t, visited)
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "DISTINCT name ORDER BY rank DESC, id ASC SEPARATOR ','")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

func TestAggregateSeparatorRejectsMalformedClauses(t *testing.T) {
	for _, expression := range []string{
		"GROUP_CONCAT(name SEPARATOR)",
		"GROUP_CONCAT(name SEPARATOR ',)",
		"GROUP_CONCAT(name SEPARATOR 1)",
		"GROUP_CONCAT(name SEPARATOR NULL)",
		"GROUP_CONCAT(name SEPARATOR other_column)",
		"GROUP_CONCAT(name SEPARATOR ',' SEPARATOR ';')",
		"GROUP_CONCAT(name SEPARATOR ',' ORDER BY rank)",
		"GROUP_CONCAT(name SEPARATOR ',', other_column)",
		"GROUP_CONCAT(name ORDER BY rank, SEPARATOR ',')",
		"GROUP_CONCAT(name ORDER BY rank SEPARATOR)",
		"GROUP_CONCAT(name ORDER BY SEPARATOR ',')",
		"GROUP_CONCAT(name SEPARATOR ',' LIMIT)",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseCallExpr(expression)
			require.Error(t, err)
			_, err = ParseQuery("SELECT " + expression + " AS names FROM records")
			require.Error(t, err)
		})
	}
}

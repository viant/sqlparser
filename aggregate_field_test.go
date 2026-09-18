package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestAggregateNullTreatment(t *testing.T) {
	for _, expression := range []string{
		"ARRAY_AGG(seller_domain IGNORE NULLS ORDER BY node_index ASC)",
		"ARRAY_AGG(DISTINCT seller_domain RESPECT NULLS ORDER BY node_index DESC LIMIT 20)",
		"ARRAY_AGG(seller_domain ignore /* comment */ nulls)",
		"IFNULL(ARRAY_AGG((nodes[SAFE_OFFSET(i)]).sellerDomain IGNORE NULLS ORDER BY i), fallback)",
	} {
		t.Run(expression, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + expression + " AS domains FROM records")
			require.NoError(t, err)
			modifiers := 0
			Traverse(q, func(n node.Node) bool {
				if value, ok := n.(*expr.NullTreatment); ok {
					modifiers++
					require.NotNil(t, value.X)
				}
				return true
			})
			require.Equal(t, 1, modifiers)
			require.Equal(t, expression, Stringify(q.List[0].Expr))
			_, err = ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
		})
	}
	for _, expression := range []string{
		"ARRAY_AGG(x IGNORE)", "ARRAY_AGG(x RESPECT n)", "ARRAY_AGG(x NULLS)",
		"ARRAY_AGG(x IGNORE NULLS RESPECT NULLS)", "ARRAY_AGG(x ORDER BY i IGNORE NULLS)",
		"ARRAY_AGG(x IGNORE NULLS LIMIT)",
	} {
		_, err := ParseCallExpr(expression)
		require.Error(t, err, expression)
	}
}

func TestComputedFieldTreeAndTransforms(t *testing.T) {
	q, err := ParseQuery("SELECT (nodes[SAFE_OFFSET(i)]).sellerDomain AS domain FROM records")
	require.NoError(t, err)
	field, ok := q.List[0].Expr.(*expr.FieldAccess)
	require.True(t, ok)
	require.Equal(t, "sellerDomain", field.Name)
	require.IsType(t, &expr.Parenthesis{}, field.X)
	column := NewColumn(q.List[0])
	require.Empty(t, column.Name, "computed field is not a direct source column")
	require.Equal(t, "domain", column.Alias)
	require.Equal(t, "(nodes[SAFE_OFFSET(i)]).sellerDomain", column.Expression)
	var fields, subscripts int
	Traverse(q, func(n node.Node) bool {
		switch n.(type) {
		case *expr.FieldAccess:
			fields++
		case *expr.Subscript:
			subscripts++
		}
		return true
	})
	require.Equal(t, 1, fields)
	require.Equal(t, 1, subscripts)
	for _, input := range []string{
		"SELECT a[0].", "SELECT (a[0]).", "SELECT a[0].`` FROM records",
	} {
		_, err := ParseQuery(input)
		require.Error(t, err, input)
	}
	input := "SELECT ARRAY_AGG((nodes[SAFE_OFFSET(i)]).sellerDomain COLLATE nocase IGNORE NULLS ORDER BY i LIMIT 20) AS domains FROM records"
	stripped, err := StripCollate(input)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "IGNORE NULLS")
	require.Contains(t, stripped, "ORDER BY i")
	require.Contains(t, stripped, "LIMIT 20")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

func TestComputedFieldQuotedEscapes(t *testing.T) {
	for _, name := range []string{
		"`foo\\`bar`", "`foo\\``", "`foo\\\\`", "`foo``bar`",
		"\"foo\\\"bar\"", "[foo\\]", "[foo]]bar]",
	} {
		t.Run(name, func(t *testing.T) {
			SQL := "SELECT a[0]." + name + " AS result FROM records WHERE id = ?"
			for _, options := range [][]Option{nil, {WithStructuralValidation()}} {
				q, err := ParseQuery(SQL, options...)
				require.NoError(t, err)
				field, ok := q.List[0].Expr.(*expr.FieldAccess)
				require.True(t, ok)
				require.Equal(t, name, field.Name)
				require.Equal(t, "result", q.List[0].Alias)
				require.Equal(t, "records", TableName(q))
				require.NotNil(t, q.Qualify)
				require.Equal(t, SQL, Stringify(q))
				again, err := ParseQuery(Stringify(q), options...)
				require.NoError(t, err)
				require.Equal(t, name, again.List[0].Expr.(*expr.FieldAccess).Name)
			}
		})
	}
	for _, name := range []string{"`foo\\`", "`foo\\", "\"foo\\\"", "[foo"} {
		t.Run("unclosed/"+name, func(t *testing.T) {
			for _, options := range [][]Option{nil, {WithStructuralValidation()}} {
				_, err := ParseQuery("SELECT a[0]."+name, options...)
				require.Error(t, err)
			}
		})
	}
}

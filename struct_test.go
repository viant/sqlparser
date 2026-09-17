package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestStructNamedFields(t *testing.T) {
	fields := "CAST(NULL AS STRING) AS feature, CAST(NULL AS FLOAT64) AS sample_count, CAST(NULL AS INT64) AS duration_hours"
	for _, name := range []string{"STRUCT", "struct", "STRUCT \n"} {
		t.Run(name, func(t *testing.T) {
			parsed, err := ParseQuery("SELECT " + name + "(" + fields + ") AS result FROM orders")
			require.NoError(t, err)
			require.Empty(t, parsed.Kind)
			call := parsed.List[0].Expr.(*expr.Call)
			require.Len(t, call.Args, 3)
			for i, alias := range []string{"feature", "sample_count", "duration_hours"} {
				field := call.Args[i].(*query.Item)
				require.Equal(t, alias, field.Alias)
				cast, err := CastExpression(field.Expr.(*expr.Call))
				require.NoError(t, err)
				require.Equal(t, "NULL", cast.Operand)
				require.Equal(t, []string{"STRING", "FLOAT64", "INT64"}[i], cast.Type)
			}
			require.Equal(t, fields, Stringify(call.Args))
			rendered := Stringify(parsed)
			again, err := ParseQuery(rendered)
			require.NoError(t, err)
			require.Equal(t, rendered, Stringify(again))
			_, direct := (Lineage{Query: parsed}).Compile().Lookup("result")
			require.False(t, direct)
		})
	}
}

func TestStructNestedFields(t *testing.T) {
	for _, expression := range []string{
		"STRUCT()", "STRUCT(1, 'abc')", "STRUCT(1 AS id, 'abc')",
		"STRUCT(1 AS `field name`, 2 AS _value)",
		"STRUCT(? /* value */ AS/* name */feature/* end */, 2 AS count)",
		"STRUCT(STRUCT(? AS value) AS nested, a[SAFE_OFFSET(0)] AS first)",
		"TO_JSON_STRING(STRUCT(CAST(NULL AS STRING) AS feature))",
		"STRUCT/*comment*/(1 AS id)",
		"TO_JSON_STRING(STRUCT /*comment*/ (1 AS id))",
		"STRUCT/*first*//*second*/(STRUCT/*inner*/(? AS value) AS nested)",
		"STRUCT-- comment\n(1 AS id)",
		"IF(1 = 1, STRUCT(1 AS value), STRUCT(CAST(NULL AS INT64) AS value))",
		"TO_JSON_STRING(STRUCT((SELECT AS STRUCT id FROM orders LIMIT 1) AS record))",
	} {
		t.Run(expression, func(t *testing.T) {
			parsed, err := ParseQuery("SELECT " + expression + " AS result FROM source")
			require.NoError(t, err)
			require.Empty(t, parsed.Kind)
			require.IsType(t, &expr.Call{}, parsed.List[0].Expr)
			rendered := Stringify(parsed)
			again, err := ParseQuery(rendered)
			require.NoError(t, err)
			require.Equal(t, rendered, Stringify(again))
		})
	}
	parsed, err := ParseQuery("SELECT AS STRUCT (1) AS value")
	require.NoError(t, err)
	require.Equal(t, "AS STRUCT", parsed.Kind)
}

func TestStructFieldsTraversalAndCollate(t *testing.T) {
	SQL := "SELECT TO_JSON_STRING(STRUCT(? AS id, STRUCT(status COLLATE nocase AS label) AS nested, " +
		"(SELECT status COLLATE nocase FROM orders LIMIT 2) AS first)) AS result FROM source"
	parsed, err := ParseQuery(SQL)
	require.NoError(t, err)
	placeholders, collations := 0, 0
	Traverse(parsed, func(n node.Node) bool {
		switch n.(type) {
		case *expr.Placeholder:
			placeholders++
		case *expr.Collate:
			collations++
		}
		return true
	})
	require.Equal(t, 1, placeholders)
	require.Equal(t, 2, collations)
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "STRUCT(? AS id, STRUCT(status AS label) AS nested")
	require.Contains(t, stripped, "LIMIT 2) AS first")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

func TestStructRejectsMalformedFields(t *testing.T) {
	for _, expression := range []string{
		"STRUCT(1 AS)", "STRUCT(1 AS , 2)", "STRUCT(1 AS 'name')", "STRUCT(1 AS a.b)",
		"STRUCT(1 AS a extra)", "STRUCT(1 AS a AS b)", "STRUCT(1 AS a,)", "STRUCT(,1 AS a)",
		"STRUCT(1 + AS a)", "STRUCT(AS a)", "STRUCT(1 implicit)",
		"STRUCT(STRUCT(1 AS) AS nested)",
		"ABS(1 AS value)", "COALESCE(1 AS value, 2)",
	} {
		_, err := ParseQuery("SELECT " + expression + " FROM source")
		require.Error(t, err, expression)
	}
}

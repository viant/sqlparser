package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
)

func TestQualifiedQuotedProjection(t *testing.T) {
	for _, tc := range []struct {
		expression, qualifier, column string
	}{
		{"selector_features.`category`", "selector_features", "`category`"},
		{"selector_features.category", "selector_features", "category"},
		{"`selector_features`.`category`", "`selector_features`", "`category`"},
		{"`selector_features`.category", "`selector_features`", "category"},
		{"selector_features.`category.name`", "selector_features", "`category.name`"},
		{"`selector.features`.`category[name]`", "`selector.features`", "`category[name]`"},
		{"selector_features.`category``name`", "selector_features", "`category``name`"},
		{"selector_features.`category\\`name`", "selector_features", "`category\\`name`"},
	} {
		for _, alias := range []struct{ source, name string }{
			{"", ""}, {" AS category", "category"}, {" category", "category"},
			{" AS `output category`", "`output category`"}, {" `output category`", "`output category`"},
		} {
			t.Run(tc.expression+alias.source, func(t *testing.T) {
				SQL := "SELECT " + tc.expression + alias.source + " FROM features AS selector_features"
				q, err := ParseQuery(SQL)
				require.NoError(t, err)
				for i := 0; i < 2; i++ {
					item := q.List[0]
					require.Equal(t, alias.name, item.Alias)
					require.Equal(t, tc.expression, Stringify(item.Expr))
					require.Equal(t, &expr.Selector{Name: tc.qualifier, X: &expr.Ident{Name: tc.column}}, item.Expr)
					q, err = ParseQuery(Stringify(q), WithStructuralValidation())
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestQualifiedQuotedDerivedTable(t *testing.T) {
	const SQL = "SELECT selector_features.`category`, selector_features.`feature_count` FROM (SELECT f.id, f.category, COUNT(*) AS feature_count FROM features f GROUP BY f.id, f.category) selector_features"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		require.Len(t, q.List, 2)
		for j, column := range []string{"`category`", "`feature_count`"} {
			require.Empty(t, q.List[j].Alias)
			require.Equal(t, "selector_features."+column, Stringify(q.List[j].Expr))
		}
		require.Equal(t, "selector_features", q.From.Alias)
		require.Contains(t, Stringify(q), "COUNT(*) AS feature_count")
		require.Contains(t, Stringify(q), "GROUP BY f.id, f.category")
		q, err = ParseQuery(Stringify(q), WithStructuralValidation())
		require.NoError(t, err)
	}
}

func TestQuotedIdentifierBoundaries(t *testing.T) {
	for _, expression := range []string{
		"`selector_features.category`", "`category[name]`", "`category``name`",
		"schema.`selector_features`.`category`", "`schema`.`selector_features`.category",
	} {
		t.Run(expression, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + expression + " FROM features")
			require.NoError(t, err)
			require.Empty(t, q.List[0].Alias)
			require.Equal(t, expression, Stringify(q.List[0].Expr))
			if expression[0] == '`' && expression != "`schema`.`selector_features`.category" {
				require.Equal(t, &expr.Ident{Name: expression}, q.List[0].Expr)
			}
		})
	}
	for _, expression := range []string{"t.`unclosed", "`t`.`unclosed", "t.`escaped\\`", "`unclosed"} {
		_, err := ParseQuery("SELECT "+expression, WithStructuralValidation())
		require.Error(t, err, expression)
	}
}

func TestSelectorPreservesQuotedBrackets(t *testing.T) {
	selector := expr.NewSelector("`group[x]`.records[x]").(*expr.Selector)
	require.Equal(t, "`group[x]`", selector.Name)
	require.Equal(t, "x", selector.Expression)
	require.Equal(t, &expr.Ident{Name: "records"}, selector.X)
}

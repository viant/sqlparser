package sqlparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func TestTableFreeUnionBranches(t *testing.T) {
	for _, SQL := range []string{
		"SELECT 1 AS id UNION ALL SELECT 2 UNION ALL SELECT 3",
		"SELECT 1 AS id UNION SELECT 2",
		"SELECT 1 AS id UNION ALL SELECT id FROM records UNION ALL SELECT 3",
		"SELECT id FROM records UNION ALL SELECT 2 UNION ALL SELECT 3",
		"SELECT 1 AS id WHERE 1 = 1 UNION ALL SELECT 2 WHERE 1 = 0 UNION ALL SELECT 3",
		"SELECT 1 AS id UNION ALL SELECT 2 ORDER BY id LIMIT 1 OFFSET 0",
		"SELECT 1 AS id /* branch */ UNION ALL /* next */ SELECT 2",
	} {
		t.Run(SQL, func(t *testing.T) {
			q, err := ParseQuery(SQL)
			require.NoError(t, err)
			branches := 0
			for branch := q; branch != nil; {
				require.Len(t, branch.List, 1)
				branches++
				if branch.Union == nil {
					break
				}
				branch = branch.Union.X
			}
			require.Equal(t, 1+strings.Count(SQL, "UNION"), branches)
			rendered := (Stringifier{PreserveWindow: true}).String(q)
			require.Equal(t, strings.Count(SQL, "UNION"), strings.Count(rendered, "UNION"))
			again, err := ParseQuery(rendered)
			require.NoError(t, err)
			require.Equal(t, rendered, (Stringifier{PreserveWindow: true}).String(again))
		})
	}
}

func TestTableFreeUnionDerivedJoin(t *testing.T) {
	var rows []string
	for i := 0; i < 10; i++ {
		row := fmt.Sprintf("SELECT %d, 'CATEGORY_%d'", 1<<i, i)
		if i == 0 {
			row = "SELECT 1 AS bit_value, 'CATEGORY_0' AS category_name"
		}
		rows = append(rows, row)
	}
	lookup := strings.Join(rows, " UNION ALL ")
	condition := "(COALESCE(o.category_bits, 0) & categories.bit_value) != 0"
	SQL := "SELECT o.id, GROUP_CONCAT(DISTINCT categories.category_name ORDER BY categories.bit_value SEPARATOR ',') AS category_names " +
		"FROM orders o LEFT JOIN (" + lookup + ") categories ON " + condition + " WHERE o.id IN (?) GROUP BY o.id"
	q, err := ParseQuery(SQL, WithStructuralValidation())
	require.NoError(t, err)
	require.Len(t, q.Joins, 1)
	join := q.Joins[0]
	require.Equal(t, "categories", join.Alias)
	require.Equal(t, condition, Stringify(join.On))
	inner := join.With.(*expr.Parenthesis).X.(*query.Select)
	require.Equal(t, "bit_value", inner.List[0].Alias)
	require.Equal(t, "category_name", inner.List[1].Alias)
	branch := inner
	for i := 0; i < 10; i++ {
		require.NotNil(t, branch)
		require.Nil(t, branch.From.X)
		require.Len(t, branch.List, 2)
		require.Equal(t, fmt.Sprint(1<<i), Stringify(branch.List[0].Expr))
		require.Equal(t, fmt.Sprintf("'CATEGORY_%d'", i), Stringify(branch.List[1].Expr))
		if i < 9 {
			require.NotNil(t, branch.Union)
			require.Equal(t, "all", branch.Union.Kind)
			branch = branch.Union.X
		} else {
			require.Nil(t, branch.Union)
		}
	}
	// Rebuild the inner SQL from its AST so retained raw text cannot hide a
	// missing UNION branch, then round-trip the complete grouped query.
	require.Equal(t, lookup, Stringify(inner))
	join.With.(*expr.Parenthesis).Raw = "(" + Stringify(inner) + ")"
	require.Equal(t, SQL, Stringify(q))
	_, err = ParseQuery(Stringify(q), WithStructuralValidation())
	require.NoError(t, err)
	for _, enclosing := range []string{
		"SELECT * FROM (" + lookup + ") categories",
		"WITH categories AS (" + lookup + ") SELECT * FROM categories",
		"SELECT ARRAY(" + lookup + ") AS categories",
	} {
		parsed, err := ParseQuery(enclosing)
		require.NoError(t, err)
		rendered := Stringify(parsed)
		require.Equal(t, strings.Fields(enclosing), strings.Fields(rendered))
		var nested *query.Select
		switch {
		case len(parsed.WithSelects) > 0:
			nested = parsed.WithSelects[0].X
		case parsed.From.X != nil:
			nested = parsed.From.X.(*expr.Raw).X.(*query.Select)
		default:
			nested = parsed.List[0].Expr.(*expr.Call).Args[0].(*query.Select)
		}
		require.Equal(t, lookup, Stringify(nested))
		_, err = ParseQuery(rendered)
		require.NoError(t, err)
	}
}

func TestTableFreeUnionRejectsIncompleteBranches(t *testing.T) {
	for _, SQL := range []string{
		"SELECT 1 UNION ALL", "SELECT 1 UNION ALL SELECT", "SELECT 1 UNION ALL SELECT 2 +",
		"SELECT 1 UNION ALL garbage", "SELECT 1 UNION ALL SELECT 2 UNION ALL",
	} {
		for _, wrapped := range []string{SQL, "SELECT * FROM (" + SQL + ") lookup", "SELECT * FROM records r JOIN (" + SQL + ") lookup ON 1 = 1"} {
			_, err := ParseQuery(wrapped)
			require.Error(t, err, wrapped)
		}
	}
}

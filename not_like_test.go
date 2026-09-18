package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestCaseNotLike(t *testing.T) {
	for _, condition := range []string{
		"name NOT LIKE '%internal%'",
		"LOWER(name) not like CONCAT('%', ?, '%')",
		"name NOT\n\tLIKE ?",
		"name NOT /* pattern */ LIKE ?",
		"name NOT LIKE(?)",
	} {
		t.Run(condition, func(t *testing.T) {
			SQL := "SELECT CASE WHEN " + condition + " THEN 1 ELSE 0 END AS allowed FROM records"
			q, err := ParseQuery(SQL)
			require.NoError(t, err)
			switchExpr, ok := q.List[0].Expr.(*expr.Switch)
			require.True(t, ok)
			predicate, ok := switchExpr.Cases[0].X.X.(*expr.Binary)
			require.True(t, ok)
			require.Equal(t, "NOT LIKE", predicate.Op)
			require.NotNil(t, predicate.X)
			require.NotNil(t, predicate.Y)
			require.Equal(t, SQL, Stringify(q))
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, q.List[0].Expr, again.List[0].Expr)
		})
	}
}

func TestNotLikePrecedenceAndTransforms(t *testing.T) {
	const SQL = "SELECT SUM(CASE WHEN name NOT LIKE $pattern AND active = $active OR fallback = $fallback THEN $yes ELSE $no END) AS total FROM records"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	condition := q.List[0].Expr.(*expr.Call).Args[0].(*expr.Switch).Cases[0].X.X.(*expr.Binary)
	require.Equal(t, "OR", condition.Op)
	and := condition.X.(*expr.Binary)
	require.Equal(t, "AND", and.Op)
	require.Equal(t, "NOT LIKE", and.X.(*expr.Binary).Op)
	require.Equal(t, "=", and.Y.(*expr.Binary).Op)
	var parameters []string
	Traverse(q, func(n node.Node) bool {
		if p, ok := n.(*expr.Placeholder); ok {
			parameters = append(parameters, p.Name)
		}
		return true
	})
	require.Equal(t, []string{"$pattern", "$active", "$fallback", "$yes", "$no"}, parameters)
	stripped, err := StripCollate("SELECT CASE WHEN name COLLATE nocase NOT LIKE ? AND active = 1 THEN 1 ELSE 0 END FROM records WHERE category NOT LIKE ?")
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "name NOT LIKE ? AND active = 1")
	require.Contains(t, stripped, "category NOT LIKE ?")
	_, err = ParseQuery(stripped, WithStructuralValidation())
	require.NoError(t, err)
}

func TestNotLikeRejectsMalformed(t *testing.T) {
	for _, condition := range []string{
		"name NOT LIKE", "NOT LIKE ?", "name NOT LIKE AND active = 1",
		"name NOTLIKE ?", "name NOT LIKE_pattern", "name NOT LIKE2",
		"name NOT LIKE ? +",
	} {
		t.Run(condition, func(t *testing.T) {
			_, err := ParseQuery("SELECT CASE WHEN " + condition + " THEN 1 ELSE 0 END FROM records")
			require.Error(t, err)
		})
	}
}

func TestNotLikeExpressionContexts(t *testing.T) {
	for _, SQL := range []string{
		"SELECT name NOT LIKE ? AS allowed FROM records",
		"SELECT name FROM records WHERE name NOT LIKE ? AND active = 1",
		"SELECT name FROM records ORDER BY name NOT LIKE ?",
		"SELECT COUNT(*) FROM records GROUP BY name NOT LIKE ?",
		"SELECT COUNT(*) FROM records HAVING MAX(name) NOT LIKE ?",
		"SELECT COALESCE(name NOT LIKE ?, FALSE) FROM records",
	} {
		t.Run(SQL, func(t *testing.T) {
			q, err := ParseQuery(SQL)
			require.NoError(t, err)
			require.Contains(t, Stringify(q), "NOT LIKE ?")
			_, err = ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
		})
	}
}

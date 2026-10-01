package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestCaseNotGlob(t *testing.T) {
	for _, condition := range []string{
		"name NOT GLOB '%internal%'",
		"LOWER(name) not glob CONCAT('%', ?, '%')",
		"name NOT\n\tGLOB ?",
		"name NOT /* pattern */ GLOB ?",
		"name NOT GLOB(?)",
	} {
		t.Run(condition, func(t *testing.T) {
			SQL := "SELECT CASE WHEN " + condition + " THEN 1 ELSE 0 END AS allowed FROM records"
			q, err := ParseQuery(SQL)
			require.NoError(t, err)
			switchExpr, ok := q.List[0].Expr.(*expr.Switch)
			require.True(t, ok)
			predicate, ok := switchExpr.Cases[0].X.X.(*expr.Binary)
			require.True(t, ok)
			require.Equal(t, "NOT GLOB", predicate.Op)
			require.NotNil(t, predicate.X)
			require.NotNil(t, predicate.Y)
			require.Equal(t, SQL, Stringify(q))
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, q.List[0].Expr, again.List[0].Expr)
		})
	}
}

func TestNotGlobPrecedenceAndTransforms(t *testing.T) {
	const SQL = "SELECT SUM(CASE WHEN name NOT GLOB $pattern AND active = $active OR fallback = $fallback THEN $yes ELSE $no END) AS total FROM records"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	condition := q.List[0].Expr.(*expr.Call).Args[0].(*expr.Switch).Cases[0].X.X.(*expr.Binary)
	require.Equal(t, "OR", condition.Op)
	and := condition.X.(*expr.Binary)
	require.Equal(t, "AND", and.Op)
	require.Equal(t, "NOT GLOB", and.X.(*expr.Binary).Op)
	require.Equal(t, "=", and.Y.(*expr.Binary).Op)
	var parameters []string
	Traverse(q, func(n node.Node) bool {
		if p, ok := n.(*expr.Placeholder); ok {
			parameters = append(parameters, p.Name)
		}
		return true
	})
	require.Equal(t, []string{"$pattern", "$active", "$fallback", "$yes", "$no"}, parameters)
	stripped, err := StripCollate("SELECT CASE WHEN name COLLATE nocase NOT GLOB ? AND active = 1 THEN 1 ELSE 0 END FROM records WHERE category NOT GLOB ?")
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "name NOT GLOB ? AND active = 1")
	require.Contains(t, stripped, "category NOT GLOB ?")
	_, err = ParseQuery(stripped, WithStructuralValidation())
	require.NoError(t, err)
}

func TestNotGlobRejectsMalformed(t *testing.T) {
	for _, condition := range []string{
		"name NOT GLOB", "NOT GLOB ?", "name NOT GLOB AND active = 1",
		"name NOTGLOB ?", "name NOT GLOB_pattern", "name NOT GLOB2",
		"name NOT GLOB ? +",
	} {
		t.Run(condition, func(t *testing.T) {
			_, err := ParseQuery("SELECT CASE WHEN " + condition + " THEN 1 ELSE 0 END FROM records")
			require.Error(t, err)
		})
	}
}

func TestNotGlobExpressionContexts(t *testing.T) {
	for _, SQL := range []string{
		"SELECT name NOT GLOB ? AS allowed FROM records",
		"SELECT name FROM records WHERE name NOT GLOB ? AND active = 1",
		"SELECT name FROM records ORDER BY name NOT GLOB ?",
		"SELECT COUNT(*) FROM records GROUP BY name NOT GLOB ?",
		"SELECT COUNT(*) FROM records HAVING MAX(name) NOT GLOB ?",
		"SELECT COALESCE(name NOT GLOB ?, FALSE) FROM records",
	} {
		t.Run(SQL, func(t *testing.T) {
			q, err := ParseQuery(SQL)
			require.NoError(t, err)
			require.Contains(t, Stringify(q), "NOT GLOB ?")
			_, err = ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
		})
	}
}

func TestGlobExpressionContexts(t *testing.T) {
	for _, operator := range []string{"GLOB", "glob", "GLOB/* pattern */"} {
		SQL := "SELECT CASE WHEN name " + operator + " '[0-9]*' AND active = 1 THEN 1 ELSE 0 END FROM records"
		q, err := ParseQuery(SQL, WithStructuralValidation())
		require.NoError(t, err)
		condition := q.List[0].Expr.(*expr.Switch).Cases[0].X.X.(*expr.Binary)
		require.Equal(t, "AND", condition.Op)
		require.Equal(t, "GLOB", condition.X.(*expr.Binary).Op)
		require.Equal(t, SQL, Stringify(q))
	}
	for _, condition := range []string{"name GLOB", "GLOB ?", "name GLOB_pattern", "name GLOB2", "name GLOB ? +"} {
		_, err := ParseQuery("SELECT CASE WHEN " + condition + " THEN 1 ELSE 0 END FROM records")
		require.Error(t, err, condition)
	}
}

// SQLite puts GLOB below equality and above AND; concatenation binds above
// multiplication. See https://www.sqlite.org/lang_expr.html#operators_and_parse_affecting_attributes
func TestGlobSQLitePrecedence(t *testing.T) {
	q, err := ParseQuery("SELECT a GLOB b = c AND active = 1 FROM records")
	require.NoError(t, err)
	and := q.List[0].Expr.(*expr.Binary)
	require.Equal(t, "AND", and.Op)
	glob := and.X.(*expr.Binary)
	require.Equal(t, "GLOB", glob.Op)
	require.Equal(t, "=", glob.Y.(*expr.Binary).Op)
	q, err = ParseQuery("SELECT a GLOB b GLOB c FROM records")
	require.NoError(t, err)
	require.Equal(t, "GLOB", q.List[0].Expr.(*expr.Binary).X.(*expr.Binary).Op)
}

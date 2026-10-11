package sqlparser

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"strings"
	"testing"
)

func TestRegexpOperatorsPreserveSourceAndPrecedence(t *testing.T) {
	for _, test := range []struct{ syntax, op string }{
		{"REGEXP", "REGEXP"}, {"regexp", "REGEXP"}, {"RLIKE", "RLIKE"},
		{"NOT REGEXP", "NOT REGEXP"}, {"not /* gap */ regexp", "NOT REGEXP"},
		{"NOT\nRLIKE", "NOT RLIKE"},
	} {
		t.Run(test.syntax, func(t *testing.T) {
			sql := "SELECT CASE WHEN name " + test.syntax + " CONCAT('(?=', $pattern, ')') AND enabled = 1 OR fallback = 1 THEN 1 ELSE 0 END AS result FROM records"
			q, err := ParseQuery(sql, WithStructuralValidation())
			require.NoError(t, err)
			condition := q.List[0].Expr.(*expr.Switch).Cases[0].X.X.(*expr.Binary)
			require.Equal(t, "OR", condition.Op)
			and := condition.X.(*expr.Binary)
			require.Equal(t, "AND", and.Op)
			require.Equal(t, test.op, and.X.(*expr.Binary).Op)
			require.Equal(t, sql, Stringify(q))
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, q.List[0].Expr, again.List[0].Expr)
		})
	}
}

func TestRegexpInNestedJoinProjection(t *testing.T) {
	const sql = `SELECT base.id FROM (SELECT id, algo_id FROM orders) base JOIN (SELECT order_id, enabled, strategies FROM config) cost ON cost.order_id = base.id AND (cost.enabled OR (cost.strategies REGEXP CONCAT('[{]', '(?=[^}]*"algorithm_ids":"([^" ]*,)?', CAST(base.algo_id AS CHAR), '(,[^" ]*)?")', '[^}]*[}]') AND base.algo_id > 0))`
	q, err := ParseQuery(sql, WithStructuralValidation())
	require.NoError(t, err)
	// The existing formatter spaces subqueries; the predicate remains intact.
	predicate := sql[strings.Index(sql, "cost.order_id ="):]
	require.Contains(t, Stringify(q), predicate)
	_, err = ParseQuery(Stringify(q), WithStructuralValidation())
	require.NoError(t, err)
}

func TestRegexpRejectsMalformedOperandsAndPreservesIdentifierPrefixes(t *testing.T) {
	for _, condition := range []string{"name REGEXP", "REGEXP ?", "name NOT REGEXP", "name RLIKE AND enabled=1", "name NOTREGEXP ?", "name REGEXP_pattern ?"} {
		_, err := ParseQuery("SELECT CASE WHEN "+condition+" THEN 1 ELSE 0 END FROM records", WithStructuralValidation())
		require.Error(t, err, condition)
	}
	for _, sql := range []string{"SELECT name AS REGEXP_pattern FROM records", "SELECT name RLIKE_label FROM records", "SELECT REGEXP_LIKE(name, 'a') AS matched FROM records"} {
		q, err := ParseQuery(sql, WithStructuralValidation())
		require.NoError(t, err)
		_, err = ParseQuery(Stringify(q), WithStructuralValidation())
		require.NoError(t, err)
	}
}

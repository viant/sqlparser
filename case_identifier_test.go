package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
)

func TestCaseKeywordIdentifierPrefixes(t *testing.T) {
	const SQL = "SELECT MAX(end_date) AS end_date FROM projection_source"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	call, ok := q.List[0].Expr.(*expr.Call)
	require.True(t, ok)
	require.Len(t, call.Args, 1)
	require.Equal(t, "end_date", Stringify(call.Args[0]))
	require.Equal(t, "end_date", q.List[0].Alias)
	require.Equal(t, SQL, Stringify(q))

	for _, expression := range []string{
		"MAX(end_date)", "MAX(END_DATE)", "MAX(end2)", "MAX(ending)",
		"MAX(when_value)", "MAX(then_value)", "MAX(else_value)", "MAX(case_value)",
		"end_date", "CAST(end_date AS DATE)", "f(value => end_date)",
		"CASE WHEN when_flag = 1 THEN then_value ELSE else_value END",
		"CASE case_type WHEN 1 THEN end_date ELSE end_backup END",
		"MAX(CASE WHEN active = 1 THEN end_date ELSE start_date END)",
		"MAX(CASE WHEN active = 1 THEN end_date ELSE start_date END/2)",
		"CASE/* condition */WHEN(x = 1)THEN(end_date)ELSE(start_date)END",
	} {
		t.Run(expression, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + expression + " AS result FROM projection_source")
			require.NoError(t, err)
			require.Equal(t, expression, Stringify(q.List[0].Expr))
			again, err := ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, q.List[0].Expr, again.List[0].Expr)
		})
	}
}

func TestCaseRequiresWholeKeywords(t *testing.T) {
	for _, expression := range []string{
		"CASE WHEN x THEN y END_date",
		"CASE WHEN x THEN_value ELSE y END",
		"CASE WHEN x THEN y ELSE_value END",
		"MAX(END)", "MAX(THEN)", "MAX(WHEN)", "MAX(ELSE)",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression + " FROM projection_source")
			require.Error(t, err)
		})
	}
}

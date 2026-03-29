package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringify_SwitchExpression(t *testing.T) {
	queryText := `SELECT CASE WHEN MAX(flag_state) = 1 THEN 'active' ELSE 'inactive' END AS status_bucket FROM t`
	queryNode, err := ParseQuery(queryText)
	require.NoError(t, err)
	require.Len(t, queryNode.List, 1)

	actual := Stringify(queryNode.List[0].Expr)
	require.Contains(t, actual, "CASE")
	require.Contains(t, actual, "WHEN")
	require.Contains(t, actual, "THEN")
	require.Contains(t, actual, "END")
}

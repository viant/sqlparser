package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripCollate(t *testing.T) {
	testCases := []struct {
		description string
		sql         string
	}{
		{
			description: "join with collate",
			sql: "SELECT * FROM a a JOIN b b ON " +
				"a.brand COLLATE utf8mb4_bin = b.brand COLLATE utf8mb4_bin AND " +
				"a.model COLLATE utf8mb4_bin = b.model COLLATE utf8mb4_bin",
		},
		{
			description: "collate in parenthesis",
			sql: "SELECT * FROM a a JOIN b b ON " +
				"(a.brand COLLATE utf8mb4_bin) = (b.brand COLLATE utf8mb4_bin)",
		},
		{
			description: "collate with function",
			sql: "SELECT * FROM a a JOIN b b ON " +
				"lower(a.brand COLLATE utf8mb4_bin) = b.brand",
		},
		{
			description: "collate in select list",
			sql:         "SELECT a.brand COLLATE utf8mb4_bin AS brand FROM a",
		},
		{
			description: "with clause",
			sql: "WITH cte AS (SELECT a.brand COLLATE utf8mb4_bin AS brand FROM a) " +
				"SELECT * FROM cte",
		},
		{
			description: "from subquery",
			sql: "SELECT * FROM " +
				"(SELECT a.brand COLLATE utf8mb4_bin AS brand FROM a) q",
		},
		{
			description: "nested from subquery",
			sql: "SELECT * FROM " +
				"(SELECT * FROM (SELECT a.brand COLLATE utf8mb4_bin AS brand FROM a) q1) q2",
		},
	}

	for _, testCase := range testCases {
		result, err := StripCollate(testCase.sql)
		require.NoError(t, err, testCase.description)
		require.NotEmpty(t, result, testCase.description)
		require.False(t, strings.Contains(strings.ToLower(result), "collate"), testCase.description)
	}
}

func TestStripCollatePreservesCollateStringLiteral(t *testing.T) {
	sql := "SELECT * FROM (SELECT 'COLLATE utf8mb4_bin' AS label FROM a) q"
	result, err := StripCollate(sql)
	require.NoError(t, err)
	require.Contains(t, result, "'COLLATE utf8mb4_bin'")
}

func TestStripCollateRejectsCollatedOpaqueSubquery(t *testing.T) {
	sql := `SELECT * FROM (SELECT a.brand COLLATE utf8mb4_bin AS brand FROM a WHERE 1=1 ${predicate.Builder().Build("AND")}) q`
	_, err := StripCollate(sql)
	require.Error(t, err)
	require.Contains(t, err.Error(), "opaque template fragments")
}

func TestStripCollatePreservesOpaqueSubqueryWithoutCollation(t *testing.T) {
	sql := `SELECT * FROM (SELECT a.brand AS brand FROM a WHERE 1=1 ${predicate.Builder().Build("AND")}) q`
	result, err := StripCollate(sql)
	require.NoError(t, err)
	require.Contains(t, result, `${predicate.Builder().Build("AND")}`)
}

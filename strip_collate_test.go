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
	}

	for _, testCase := range testCases {
		result, err := StripCollate(testCase.sql)
		require.NoError(t, err, testCase.description)
		require.NotEmpty(t, result, testCase.description)
		require.False(t, strings.Contains(strings.ToLower(result), "collate"), testCase.description)
	}
}

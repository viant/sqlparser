package source

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptStructure(t *testing.T) {
	for _, SQL := range []string{
		"a[b[0]]", "a[0][1]", "a[IF(x = ']', 0, 1)]", "a[0 /* ] */]",
		"a[0 -- ]\n]", "(a)[0]", "`a`[0]", "a [alias]]name]", "t.[column]",
	} {
		t.Run(SQL, func(t *testing.T) { require.NoError(t, ValidateStructure(SQL)) })
	}
	for _, SQL := range []string{"a[0", "a[b[0]", "a[(0])", "a[0]]", "a [alias]]"} {
		t.Run(SQL, func(t *testing.T) { require.Error(t, ValidateStructure(SQL)) })
	}
	SQL := "a[b[0] + IF(x = ']', 0, 1)] tail"
	group, end, ok := ReadGroupString(SQL, 1, '[', ']')
	require.True(t, ok)
	require.Equal(t, "[b[0] + IF(x = ']', 0, 1)]", group)
	require.Equal(t, " tail", SQL[end:])
	group, _, ok = ReadGroupString("(a[b[0]], a[IF(x = ')', 0, 1)]) tail", 0, '(', ')')
	require.True(t, ok)
	require.Equal(t, "(a[b[0]], a[IF(x = ')', 0, 1)])", group)
}

func TestSubscriptSourceBoundaries(t *testing.T) {
	require.Equal(t, []string{"a[b[0], 1]", "id"}, SplitTopLevelCSV("a[b[0], 1], id"))
	SQL := "a[offset] + a[b[0]] OFFSET 2"
	require.Equal(t, strings.LastIndex(SQL, "OFFSET"), FindTopLevelKeyword(SQL, "offset", 0))
	require.Equal(t, strings.LastIndex(SQL, "OFFSET"), CriteriaBoundary(SQL))
	core, alias := SplitTopLevelAlias("a[b[0] + 1] [alias]")
	require.Equal(t, "a[b[0] + 1]", core)
	require.Equal(t, "[alias]", alias)
	SQL = "a[$INDEX + IF(x = '$INDEX]', 0, 1)] [$INDEX]"
	require.Equal(t, "a[? + IF(x = '$INDEX]', 0, 1)] [$INDEX]", Token("$INDEX").ReplaceAll(SQL, "?"))
	require.Equal(t, "a[0     \n] [--]", MaskLineComments("a[0 -- ]\n] [--]"))
}

func TestBracketQuotesAfterKeywords(t *testing.T) {
	for _, keyword := range []string{"AS", "FROM", "SELECT", "JOIN", "WHERE", "GROUP BY", "ORDER /* hint */ BY", "GROUP -- hint\nBY", "INTO", "UPDATE", "as", "from"} {
		for _, quoted := range []string{"[a--b]", "[a]]b]", "[$INDEX]"} {
			SQL := keyword + quoted
			t.Run(SQL, func(t *testing.T) {
				require.NoError(t, ValidateStructure(SQL))
				require.Equal(t, strings.ReplaceAll(SQL, "-- hint", "       "), MaskLineComments(SQL))
				require.Equal(t, SQL, Token("$INDEX").ReplaceAll(SQL, "?"))
				require.Equal(t, "SQL quoted text", ProtectionAt(SQL, len(keyword)+1))
			})
		}
	}
	for _, base := range []string{"array", "from_array", "t.from", "$FROM", ":FROM", "@FROM", "`FROM`", "(a)", "a[0]"} {
		SQL := base + "[$INDEX]"
		t.Run(SQL, func(t *testing.T) {
			require.Equal(t, base+"[?]", Token("$INDEX").ReplaceAll(SQL, "?"))
		})
	}
}

func TestContextualKeywordSubscriptTokens(t *testing.T) {
	for _, SQL := range []string{
		"SELECT values[$INDEX] FROM src", "SELECT a[values[$INDEX]] FROM src",
		"SELECT by[$INDEX] FROM src", "SELECT update[$INDEX] FROM src",
		"SELECT /* hint */ update[$INDEX] FROM src", "SELECT -- hint\nupdate[$INDEX] FROM src",
		"SELECT table[$INDEX] FROM src", "SELECT a[returning[$INDEX]] FROM src",
	} {
		t.Run(SQL, func(t *testing.T) {
			require.NoError(t, ValidateStructure(SQL))
			require.Equal(t, strings.ReplaceAll(SQL, "$INDEX", "?"), Token("$INDEX").ReplaceAll(SQL, "?"))
		})
	}
}

package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
)

func TestMySQLMatchAgainst(t *testing.T) {
	for _, search := range []string{
		"?", ":Query IN NATURAL LANGUAGE MODE", "'hotel' IN BOOLEAN MODE",
		"'hotel' WITH QUERY EXPANSION", "'hotel' IN NATURAL LANGUAGE MODE WITH QUERY EXPANSION",
		"CONCAT('IN BOOLEAN MODE', ?) IN /* mode */ NATURAL LANGUAGE MODE",
	} {
		t.Run(search, func(t *testing.T) {
			expression := "MATCH(d.title,d.body) AGAINST(" + search + ")"
			SQL := "SELECT " + expression + " AS score FROM docs d WHERE d.active = 1 AND " + expression + " ORDER BY " + expression + " DESC LIMIT 10000"
			parsed, err := ParseQuery(SQL, WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, strings.Replace(expression, "AGAINST(", "AGAINST (", 1), Stringify(parsed.List[0].Expr))
			binary := parsed.List[0].Expr.(*expr.Binary)
			require.Equal(t, "AGAINST", binary.Op)
			require.Len(t, binary.X.(*expr.Call).Args, 2)
			require.NotNil(t, binary.Y.(*expr.Parenthesis).X)
			again, err := ParseQuery(Stringify(parsed), WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, Stringify(parsed), Stringify(again))
			_, err = ParseQuery("SELECT x.* FROM (" + SQL + ") x")
			require.NoError(t, err)
		})
	}
}

func TestMySQLMatchAgainstRejectsMalformed(t *testing.T) {
	for _, expression := range []string{
		"MATCH() AGAINST(?)", "MATCH(a) AGAINST()", "MATCH(a) AGAINST",
		"MATCH(a) AGAINST(? IN RANDOM MODE)", "MATCH(a) AGAINST(? IN BOOLEAN MODE garbage)",
		"MATCH(a) AGAINST(? IN BOOLEAN MODE WITH QUERY EXPANSION)", "MATCH(a) AGAINST(?, ?)",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression + " AS score FROM docs")
			require.Error(t, err)
		})
	}
}

package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
	"testing"
)

func TestQuotedAliasNestedProjection(t *testing.T) {
	for _, alias := range []string{`"pseudo_column"`, "`pseudo_column`", "pseudo_column"} {
		t.Run(alias, func(t *testing.T) {
			q, err := ParseQuery("SELECT n.* FROM (SELECT r.*, '' AS " + alias + " FROM records r) n")
			if err != nil {
				t.Fatal(err)
			}
			raw, ok := q.From.X.(*expr.Raw)
			if !ok {
				t.Fatalf("source %T", q.From.X)
			}
			inner, ok := raw.X.(*query.Select)
			if !ok {
				t.Fatalf("inner %T", raw.X)
			}
			if len(inner.List) != 2 || inner.From.X == nil {
				t.Fatalf("incomplete inner: %#v SQL=%s", inner, Stringify(inner))
			}
		})
	}
}

func TestQuotedAliasSpellingAndRoundTrip(t *testing.T) {
	for _, alias := range []string{`"pseudo column"`, `"pseudo""column"`, "`pseudo``column`", "[pseudo]]column]", `"order"`} {
		for _, as := range []string{" AS ", " "} {
			SQL := "SELECT ''" + as + alias + ", 1 AS id FROM records WHERE id = 1"
			parsed, err := ParseQuery(SQL)
			if err != nil {
				t.Fatalf("%s: %v", SQL, err)
			}
			if len(parsed.List) != 2 || parsed.List[0].Alias != alias || parsed.From.X == nil || parsed.Qualify == nil {
				t.Fatalf("lost structure: %s -> %s", SQL, Stringify(parsed))
			}
			again, err := ParseQuery(Stringify(parsed))
			if err != nil || len(again.List) != 2 || again.List[0].Alias != alias || again.From.X == nil {
				t.Fatalf("roundtrip %s: %v", SQL, err)
			}
		}
	}
}

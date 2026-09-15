package sqlparser

import (
	"fmt"
	"testing"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func TestValidAliasCanonicalIdentifiers(t *testing.T) {
	for _, alias := range []string{"value", "_value", "_", "__value2", "éclair", "_値2", "from_値", "from値", "from$value", "as値", "as$value", "on値", "join値", "where値", "having値", "window値", "union値", `"_値"`, `"_a""b"`, "`_a``b`", "[_a]]b]"} {
		for _, as := range []string{" AS ", " "} {
			for _, context := range []string{"projection", "source", "join", "nested", "cte"} {
				t.Run(alias+as+context, func(t *testing.T) {
					var sql string
					switch context {
					case "projection":
						sql = "SELECT 1" + as + alias + ", 2 AS control FROM records WHERE id = 1"
					case "source":
						sql = "SELECT * FROM records" + as + alias + " WHERE id = 1"
					case "join":
						sql = "SELECT * FROM records r JOIN records" + as + alias + " ON id = 1"
					case "nested":
						sql = "SELECT probe.* FROM (SELECT 1" + as + alias + ") probe"
					case "cte":
						sql = "WITH c AS (SELECT 1" + as + alias + ") SELECT * FROM c"
					}
					for i := 0; i < 2; i++ {
						parsed, err := ParseQuery(sql)
						if err != nil {
							t.Fatalf("%s: %v", sql, err)
						}
						var got string
						switch context {
						case "projection":
							if len(parsed.List) != 2 || parsed.From.X == nil || parsed.Qualify == nil {
								t.Fatalf("lost remainder: %s", Stringify(parsed))
							}
							got = parsed.List[0].Alias
						case "source":
							got = parsed.From.Alias
							if parsed.Qualify == nil {
								t.Fatal("lost WHERE")
							}
						case "join":
							if len(parsed.Joins) != 1 || parsed.Joins[0].On == nil {
								t.Fatalf("lost JOIN: %s", Stringify(parsed))
							}
							got = parsed.Joins[0].Alias
						case "nested":
							raw, ok := parsed.From.X.(*expr.Raw)
							if !ok {
								t.Fatalf("source %T", parsed.From.X)
							}
							inner, ok := raw.X.(*query.Select)
							if !ok || len(inner.List) != 1 {
								t.Fatalf("inner %T", raw.X)
							}
							got = inner.List[0].Alias
						case "cte":
							if len(parsed.WithSelects) != 1 || len(parsed.WithSelects[0].X.List) != 1 {
								t.Fatalf("lost CTE: %s", Stringify(parsed))
							}
							got = parsed.WithSelects[0].X.List[0].Alias
						}
						if got != alias {
							t.Fatalf("alias %q, want raw %q", got, alias)
						}
						sql = Stringify(parsed)
					}
				})
			}
		}
	}
}

func TestValidAliasUnderscoreOuterCAST(t *testing.T) {
	sql := "SELECT probe.*, CAST(probe._value AS int) FROM (SELECT 1 AS _value) probe"
	for i := 0; i < 2; i++ {
		parsed, err := ParseQuery(sql)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.List) != 2 {
			t.Fatalf("lost projection: %s", Stringify(parsed))
		}
		call, ok := parsed.List[1].Expr.(*expr.Call)
		if !ok {
			t.Fatalf("CAST %T", parsed.List[1].Expr)
		}
		if got := Stringify(call); got != "CAST(probe._value AS int)" {
			t.Fatalf("CAST changed: %s", got)
		}
		inner := parsed.From.X.(*expr.Raw).X.(*query.Select)
		if inner.List[0].Alias != "_value" {
			t.Fatal("inner alias changed")
		}
		sql = Stringify(parsed)
	}
}

func TestValidAliasMalformedSuffixes(t *testing.T) {
	for _, alias := range []string{"_value!", "_value.name", "_value-name", "値!", "from値!", "from$value!", "_value\x00", "_value\xff", `"_value"suffix`} {
		for _, as := range []string{" AS ", " "} {
			for _, format := range []string{"SELECT 1%s", "SELECT 1%s FROM records", "SELECT * FROM records%s WHERE id=1", "SELECT * FROM (SELECT 1%s) probe"} {
				sql := fmt.Sprintf(format, as+alias)
				if parsed, err := ParseQuery(sql); err == nil {
					t.Errorf("malformed suffix accepted: %q -> %s", sql, Stringify(parsed))
				}
			}
		}
	}
}

// Exercise clause-prefix handling at the alias owner independently of operand
// syntax (for example, a selector followed by EXCEPT).
func TestValidAliasClausePrefixes(t *testing.T) {
	for _, prefix := range []string{"as", "on", "from", "join", "where", "having", "window", "union", "except"} {
		for _, suffix := range []string{"_value", "値", "$value"} {
			alias := prefix + suffix
			for _, as := range []string{" AS ", " "} {
				cursor := parsly.NewCursor("", []byte(as+alias+" FROM records"), 0)
				got, err := discoverAlias(cursor)
				if err != nil || got != alias {
					t.Errorf("%q: alias=%q error=%v", as+alias, got, err)
				}
			}
		}
	}
}

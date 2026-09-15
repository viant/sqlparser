package sqlparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/source"
)

func TestMalformedAliasReviewCases(t *testing.T) {
	for _, sql := range []string{
		`SELECT '' AS "bad FROM records`,
		`SELECT '' "bad FROM records`,
		`SELECT '' AS [bad FROM records`,
		`SELECT '' AS FROM records`,
		`SELECT '' AS "x"y FROM records`,
	} {
		t.Run(sql, func(t *testing.T) {
			parsed, err := ParseQuery(sql)
			if err == nil {
				t.Fatalf("malformed alias returned partial success: %s", Stringify(parsed))
			}
		})
	}
}

func TestMalformedAliasNativeOwners(t *testing.T) {
	for _, alias := range []string{
		`"bad`, "`bad", `[bad`, `"x"y`, "`x`y", `[x]y`,
		`bad-name`, `bad.name`, `bad!`, `123bad`,
	} {
		for _, as := range []string{" AS ", " "} {
			for _, format := range []string{
				"SELECT ''%s FROM records WHERE id = 1",
				"SELECT id + 1%s FROM records WHERE id = 1",
				"SELECT 1%s",
				"SELECT id FROM records UNION ALL SELECT id%s FROM records",
				"SELECT n.* FROM (SELECT ''%s FROM records) n",
				"WITH c AS (SELECT ''%s FROM records) SELECT * FROM c",
				"SELECT * FROM records%s WHERE id = 1",
				"SELECT r.* FROM records r JOIN records%s ON r.id = x.id",
			} {
				sql := fmt.Sprintf(format, as+alias)
				t.Run(sql, func(t *testing.T) {
					if parsed, err := ParseQuery(sql); err == nil {
						t.Fatalf("malformed alias returned partial success: %s", Stringify(parsed))
					}
				})
			}
		}
	}
	for _, tail := range []string{"", " ", ", 1 FROM records", "FROM records", "WHERE id = 1", "JOIN records", "AS x", "123", "@bad", "'bad'"} {
		sql := "SELECT '' AS " + tail
		t.Run(sql, func(t *testing.T) {
			if parsed, err := ParseQuery(sql); err == nil {
				t.Fatalf("missing/invalid explicit alias returned success: %s", Stringify(parsed))
			}
		})
	}
	if list, err := ParseList(`'' AS "x"y, id`); err == nil {
		t.Fatalf("ParseList accepted malformed alias: %+v", list)
	}
}

func TestMalformedAliasStructuralValidationScope(t *testing.T) {
	for _, sql := range []string{`SELECT '' AS FROM records`, `SELECT '' AS "x"y FROM records`, `SELECT '' AS bad-name FROM records`} {
		if err := source.ValidateStructure(sql); err != nil {
			t.Fatalf("balanced control rejected by structural validator: %v", err)
		}
		if parsed, err := ParseQuery(sql, WithStructuralValidation()); err == nil {
			t.Fatalf("balanced malformed alias returned partial success: %s", Stringify(parsed))
		}
	}
}

func TestMalformedAliasValidControls(t *testing.T) {
	for _, alias := range []string{`plain_name2`, `from_records`, `as_value`, `limit_value`, `"pseudo column"`, `"pseudo""column"`, "`pseudo``column`", `[pseudo]]column]`, `"order"`} {
		for _, as := range []string{" AS ", " "} {
			sql := "WITH c AS (SELECT id + 1" + as + alias + ", CAST(id AS DECIMAL(10, 2)) AS amount, CASE WHEN id = 1 THEN 2 ELSE 3 END AS flag FROM records WHERE id = 1) SELECT c.* FROM c"
			t.Run(sql, func(t *testing.T) {
				for i := 0; i < 2; i++ {
					parsed, err := ParseQuery(sql)
					if err != nil {
						t.Fatal(err)
					}
					if len(parsed.WithSelects) != 1 || parsed.From.X == nil {
						t.Fatalf("lost CTE/source: %s", Stringify(parsed))
					}
					inner := parsed.WithSelects[0].X
					if len(inner.List) != 3 || inner.List[0].Alias != alias || inner.From.X == nil || inner.Qualify == nil {
						t.Fatalf("lost alias/remainder: %s", Stringify(inner))
					}
					if _, ok := inner.List[0].Expr.(*expr.Binary); !ok {
						t.Fatalf("binary owner: %T", inner.List[0].Expr)
					}
					if _, ok := inner.List[1].Expr.(*expr.Call); !ok {
						t.Fatalf("CAST owner: %T", inner.List[1].Expr)
					}
					if _, ok := inner.List[2].Expr.(*expr.Switch); !ok {
						t.Fatalf("CASE owner: %T", inner.List[2].Expr)
					}
					sql = Stringify(parsed)
				}
			})
		}
	}
}

func TestMalformedAliasTemplateHookControl(t *testing.T) {
	calls := 0
	sql := `WITH c AS (SELECT id AS "x" FROM records AS r @suffix WHERE 1=1 ${predicate.Builder().Build("AND")} ORDER BY id) SELECT c.* FROM c`
	parsed, err := ParseQuery(sql, WithErrorHandler(func(err error, cursor *parsly.Cursor, dest any) error {
		from, ok := dest.(*query.From)
		if !ok || !strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "@suffix") {
			return err
		}
		calls++
		from.Unparsed = "@suffix"
		cursor.Pos += len("@suffix")
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	inner := parsed.WithSelects[0].X
	if calls != 1 || inner.From.Alias != "r" || inner.From.Unparsed != "@suffix" || len(inner.OrderBy) != 1 || !strings.Contains(Stringify(inner), `${predicate.Builder().Build("AND")}`) {
		t.Fatalf("lost nested hook/expression/clause: calls=%d SQL=%s", calls, Stringify(parsed))
	}
}

func TestMalformedAliasAdditionalConsumers(t *testing.T) {
	for _, clause := range []string{"GROUP BY", "ORDER BY"} {
		sql := "SELECT id FROM records " + clause + ` id + 1 AS "x"y`
		if parsed, err := ParseQuery(sql); err == nil {
			t.Fatalf("%s lost alias error: %s", clause, Stringify(parsed))
		}
	}
	if parsed, err := ParseDelete(`DELETE r FROM records r JOIN records AS "x"y ON r.id = x.id`); err == nil {
		t.Fatalf("DELETE join lost alias error: %s", Stringify(parsed))
	}
}

func TestMalformedAliasTokenBoundaryControls(t *testing.T) {
	for _, sql := range []string{
		`SELECT 1 AS "x"`,
		`SELECT 1 AS"x" FROM records`,
		"SELECT 1 `x`",
		`SELECT 1 [x]`,
		`SELECT 1 AS x`,
		`SELECT 1 AS "x", 2 AS y FROM records AS "r" WHERE id = 1`,
		`SELECT 1 AS "x"/* alias comment */ FROM records r JOIN records AS [j] ON r.id = j.id`,
		`SELECT 1 AS "x"-- alias comment
 FROM records r`,
	} {
		t.Run(sql, func(t *testing.T) {
			parsed, err := ParseQuery(sql)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.List) == 0 || parsed.List[0].Alias == "" {
				t.Fatalf("lost alias: %s", Stringify(parsed))
			}
			if strings.Contains(sql, "FROM") && parsed.From.X == nil {
				t.Fatalf("lost FROM: %s", Stringify(parsed))
			}
			if strings.Contains(sql, "JOIN") && (len(parsed.Joins) != 1 || parsed.Joins[0].Alias != "[j]" || parsed.Joins[0].On == nil) {
				t.Fatalf("lost JOIN: %s", Stringify(parsed))
			}
		})
	}
}

func TestMalformedAliasScalarSubquery(t *testing.T) {
	for _, alias := range []string{`"bad`, "`bad", `[bad`, `"x"y`, `bad-name`, `FROM`} {
		sql := "SELECT (SELECT 1 AS " + alias + " FROM records) AS value"
		// An unclosed protected region may prevent the outer parenthesis
		// matcher from reaching the subquery at all. That is the existing
		// opt-in structural validator's responsibility.
		if parsed, err := ParseQuery(sql, WithStructuralValidation()); err == nil {
			t.Fatalf("malformed scalar subquery returned success: %s", Stringify(parsed))
		}
		if source.ValidateStructure(sql) == nil {
			if parsed, err := ParseQuery(sql); err == nil {
				t.Fatalf("balanced malformed scalar alias returned success: %s", Stringify(parsed))
			}
		}
	}
}

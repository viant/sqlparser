package sqlparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestSelectAliasBoundaryReviewCases(t *testing.T) {
	for _, sql := range []string{
		`SELECT 1 AS x y FROM records`,
		`SELECT 1 x y FROM records`,
		`SELECT 1 AS "x" y FROM records`,
	} {
		t.Run(sql, func(t *testing.T) {
			parsed, err := ParseQuery(sql)
			if err == nil {
				t.Fatalf("extra alias token accepted: %s", Stringify(parsed))
			}
		})
	}
}

func TestSelectAliasBoundaryMalformedRemainders(t *testing.T) {
	for _, alias := range []string{"x", `"x"`, "`x`", "[x]"} {
		for _, as := range []string{" AS ", " "} {
			for _, remainder := range []string{" y", ` "y"`, " `y`", " [y]", "\t y", " /* hint */ y", " /* one */ /* two */ y", " -- hint\n y", " from_records", " limit_value", " limit_value!", " limit!", " order_by", " + 2"} {
				for _, format := range []string{
					"SELECT 1%s FROM records",
					"SELECT id + 1%s FROM records",
					"SELECT COUNT(*)%s FROM records",
					"SELECT 1 /* before alias */%s FROM records",
					"SELECT 1%s, 2 AS z FROM records",
					"WITH c AS (SELECT 1%s FROM records) SELECT * FROM c",
					"SELECT n.* FROM (SELECT 1%s FROM records) n",
				} {
					sql := fmt.Sprintf(format, as+alias+remainder)
					t.Run(sql, func(t *testing.T) {
						parsed, err := ParseQuery(sql)
						if err == nil {
							t.Fatalf("extra alias token accepted: %s", Stringify(parsed))
						}
					})
				}
			}
		}
	}
	if list, err := ParseList(`1 AS x y, 2 AS z`); err == nil {
		t.Fatalf("ParseList accepted extra alias: %+v", list)
	}
}

func TestSelectAliasBoundaryValidClauses(t *testing.T) {
	for _, alias := range []string{"x", `"x"`, "`x`", "[x]", `"x""y"`, "`x``y`", "[x]]y]"} {
		for _, sep := range []string{" ", " /* hint */ ", " /* one */ /* two */ ", " -- hint\n "} {
			sql := "SELECT id + 1 AS " + alias + sep + ", COUNT(*) AS total FROM records r WHERE id > 0 GROUP BY id HAVING COUNT(*) > 0 ORDER BY id LIMIT 10 OFFSET 2"
			t.Run(sql, func(t *testing.T) {
				for i := 0; i < 2; i++ {
					parsed, err := ParseQuery(sql)
					if err != nil {
						t.Fatal(err)
					}
					if len(parsed.List) != 2 || parsed.List[0].Alias != alias || parsed.From.X == nil || parsed.Qualify == nil || len(parsed.GroupBy) != 1 || parsed.Having == nil || len(parsed.OrderBy) != 1 || parsed.Limit == nil || parsed.Offset == nil {
						t.Fatalf("lost alias/clause: %s", Stringify(parsed))
					}
					if _, ok := parsed.List[0].Expr.(*expr.Binary); !ok {
						t.Fatalf("lost binary owner: %T", parsed.List[0].Expr)
					}
					sql = (Stringifier{PreserveWindow: true}).String(parsed)
				}
			})
		}
	}
	for _, sql := range []string{
		`SELECT 1 AS x`, `SELECT 1 x`, `SELECT 1 AS"x"`,
		"SELECT 1 AS x FROM`records`",
		`SELECT id /* before operator */ + 1 AS x FROM records`,
		`SELECT id /* before alias */ AS x FROM records`,
		`SELECT id + 1 AS x /* hint */ FROM records`,
		`SELECT id + 1 AS x FROM records UNION ALL SELECT id + 2 AS x FROM records`,
	} {
		parsed, err := ParseQuery(sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if len(parsed.List) != 1 || parsed.List[0].Alias == "" {
			t.Fatalf("lost projection: %s", Stringify(parsed))
		}
		if strings.Contains(sql, "FROM") && parsed.From.X == nil {
			t.Fatalf("lost FROM: %s", Stringify(parsed))
		}
		if strings.Contains(sql, "UNION") && parsed.Union == nil {
			t.Fatalf("lost UNION: %s", Stringify(parsed))
		}
	}
}

func TestSelectAliasBoundaryFromExtensions(t *testing.T) {
	for _, suffix := range []string{
		`@suffix ORDER BY id LIMIT 10`,
		`QUALIFY ROW_NUMBER() OVER (PARTITION BY id ORDER BY id) = 1`,
	} {
		calls := 0
		sql := `SELECT id AS "x" FROM records r ` + suffix
		parsed, err := ParseQuery(sql, WithErrorHandler(func(err error, cursor *parsly.Cursor, dest any) error {
			from, ok := dest.(*query.From)
			if !ok {
				return err
			}
			remaining := string(cursor.Input[cursor.Pos:])
			if strings.HasPrefix(remaining, "@suffix") {
				from.Unparsed = "@suffix"
				cursor.Pos += len("@suffix")
			} else if strings.HasPrefix(remaining, "QUALIFY ") {
				from.Unparsed = remaining
				cursor.Pos = len(cursor.Input)
			} else {
				return err
			}
			calls++
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || parsed.From.Unparsed == "" || parsed.List[0].Alias != `"x"` {
			t.Fatalf("lost extension: %s", Stringify(parsed))
		}
		if strings.HasPrefix(suffix, "@") && (len(parsed.OrderBy) != 1 || parsed.Limit == nil) {
			t.Fatalf("lost following clause: %s", Stringify(parsed))
		}
	}
}

func TestSelectAliasBoundaryProjectionExtension(t *testing.T) {
	for _, ending := range []string{" FROM records", ", 2 AS z FROM records", ""} {
		calls := 0
		parsed, err := ParseQuery("SELECT 1 AS x @extension"+ending, WithErrorHandler(func(err error, cursor *parsly.Cursor, dest any) error {
			item, ok := dest.(*query.Item)
			if !ok || !strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "@extension") {
				return err
			}
			item.Meta = "@extension"
			cursor.Pos += len("@extension")
			calls++
			return nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || parsed.List[0].Alias != "x" || parsed.List[0].Meta != "@extension" {
			t.Fatalf("extension owner lost: %+v", parsed)
		}
		if strings.Contains(ending, "FROM") && parsed.From.X == nil {
			t.Fatal("extension dropped FROM")
		}
		if strings.HasPrefix(ending, ",") && len(parsed.List) != 2 {
			t.Fatal("extension dropped next projection")
		}
	}
	for _, sql := range []string{"SELECT 1 AS x y FROM records", "SELECT 1 AS x @extension y FROM records"} {
		calls := 0
		parsed, err := ParseQuery(sql, WithErrorHandler(func(err error, cursor *parsly.Cursor, dest any) error {
			calls++
			if strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "@extension") {
				cursor.Pos += len("@extension")
			}
			return nil
		}))
		if err == nil {
			t.Fatalf("handler without progress discarded alias remainder: %s", Stringify(parsed))
		}
		if calls == 0 {
			t.Fatal("boundary did not consult extension handler")
		}
	}
}

func TestSelectAliasBoundaryOpaqueDialectBody(t *testing.T) {
	for _, body := range []string{
		`SELECT ARRAY_AGG(id ORDER BY id LIMIT 1)[OFFSET(0)] AS x FROM records WHERE _TABLE_SUFFIX = '2020'`,
		`SELECT ARRAY_AGG(id ORDER BY id LIMIT 1)[OFFSET(0)] + 1 AS x FROM records`,
		`SELECT ROW_NUMBER() OVER (PARTITION BY id ORDER BY id) AS x FROM records QUALIFY x = 1`,
		`SELECT ROW_NUMBER() OVER /* window hint */ (PARTITION BY id ORDER BY id) AS x FROM records QUALIFY x = 1`,
	} {
		sql := "WITH c AS (" + body + ") SELECT c.* FROM c"
		parsed, err := ParseQuery(sql)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.WithSelects) != 1 || parsed.WithSelects[0].Raw != "("+body+")" || parsed.From.X == nil {
			t.Fatalf("lost opaque CTE body: %s", Stringify(parsed))
		}
		inner := parsed.WithSelects[0].X
		if len(inner.List) == 0 {
			t.Fatal("lost native projection")
		}
		callFound := false
		Traverse(inner.List[0].Expr, func(n node.Node) bool {
			if _, ok := n.(*expr.Call); ok {
				callFound = true
			}
			return true
		})
		if !callFound {
			t.Fatalf("lost native call owner: %T", inner.List[0].Expr)
		}
		if !strings.Contains(Stringify(parsed), body) {
			t.Fatalf("opaque dialect body changed: %s", Stringify(parsed))
		}
	}
}

package sqlparser

import (
	"strings"
	"testing"

	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/query"
)

func TestSelectionKindASTAndRoundTrip(t *testing.T) {
	for _, tc := range []struct{ name, SQL, kind string }{
		{"distinct", "SELECT DISTINCT r.id, r.name FROM records r WHERE r.id > 0", "DISTINCT"},
		{"lowercase distinct", "select distinct r.id from records r", "distinct"},
		{"all", "SELECT ALL r.id, r.name FROM records r", "ALL"},
		{"lowercase all", "select all r.id from records r", "all"},
		{"ordinary", "SELECT r.id, r.name FROM records r", ""},
		{"table free distinct", "SELECT DISTINCT 1 AS id", "DISTINCT"},
		{"ordinary identifier prefixes", "SELECT all_records, distinctive FROM records", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseQuery(tc.SQL)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "ordinary identifier prefixes" && (NewColumn(parsed.List[0]).Identity() != "all_records" || NewColumn(parsed.List[1]).Identity() != "distinctive") {
				t.Fatalf("ordinary identifiers were consumed: %s", Stringify(parsed))
			}
			if parsed.Kind != tc.kind {
				t.Fatalf("native Select.Kind=%q, want %q", parsed.Kind, tc.kind)
			}
			for _, render := range []func() string{
				func() string { return Stringify(parsed) },
				func() string { return (Stringifier{PreserveWindow: true}).String(parsed) },
			} {
				SQL := render()
				again, err := ParseQuery(SQL)
				if err != nil {
					t.Fatal(err)
				}
				if again.Kind != tc.kind {
					t.Fatalf("round trip lost modifier: %s; kind=%q", SQL, again.Kind)
				}
				if len(again.List) != len(parsed.List) {
					t.Fatalf("projection changed: %s", SQL)
				}
				for i, item := range parsed.List {
					if NewColumn(item).Identity() != NewColumn(again.List[i]).Identity() || Stringify(item.Expr) != Stringify(again.List[i].Expr) {
						t.Fatalf("projection changed: %s", SQL)
					}
				}
				if Stringify(again) != Stringify(parsed) {
					t.Fatalf("rendering is not stable: %s", SQL)
				}
			}
		})
	}
}

func TestSelectionKindScopesAndASTRendering(t *testing.T) {
	parsed, err := ParseQuery("SELECT ALL r.id FROM (SELECT DISTINCT id FROM records) r")
	if err != nil {
		t.Fatal(err)
	}
	inner := parsed.NestedSelect()
	if parsed.Kind != "ALL" || inner == nil || inner.Kind != "DISTINCT" {
		t.Fatalf("outer=%q inner=%+v", parsed.Kind, inner)
	}
	SQL := Stringify(parsed)
	again, err := ParseQuery(SQL)
	if err != nil || again.Kind != "ALL" || again.NestedSelect() == nil || again.NestedSelect().Kind != "DISTINCT" {
		t.Fatalf("nested round trip: %s; %v", SQL, err)
	}
	// Render the nested AST itself, without relying on retained Raw SQL.
	innerSQL := Stringify(inner)
	if !strings.HasPrefix(innerSQL, "SELECT DISTINCT ") {
		t.Fatal(innerSQL)
	}
	innerAgain, err := ParseQuery(innerSQL)
	if err != nil || innerAgain.Kind != "DISTINCT" {
		t.Fatalf("inner AST round trip: %s; %v", innerSQL, err)
	}
	for _, kind := range []string{"DISTINCT", "ALL", ""} {
		statement := &query.Select{Kind: kind, List: query.List{query.NewItem(expr.NewIntLiteral("1"))}}
		SQL := Stringify(statement)
		got, err := ParseQuery(SQL)
		if err != nil || got.Kind != kind {
			t.Fatalf("AST rendering %q: %v %+v", SQL, err, got)
		}
	}
}

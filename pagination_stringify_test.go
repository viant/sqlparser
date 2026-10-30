package sqlparser

import "testing"

func TestStringifyPreservesPagination(t *testing.T) {
	for _, sql := range []string{"SELECT id FROM users LIMIT 1000", "SELECT id FROM users LIMIT 10 OFFSET 2", "WITH users AS (SELECT id FROM source LIMIT 3) SELECT id FROM users LIMIT 2"} {
		t.Run(sql, func(t *testing.T) {
			query, err := ParseQuery(sql)
			if err != nil {
				t.Fatal(err)
			}
			actual := (Stringifier{PreserveWindow: true}).String(query)
			if actual != sql {
				t.Fatalf("stringified %q, want %q", actual, sql)
			}
		})
	}
}

func TestDefaultStringifyRetainsWindowOmission(t *testing.T) {
	query, err := ParseQuery("SELECT id FROM users LIMIT 10 OFFSET 2")
	if err != nil {
		t.Fatal(err)
	}
	if actual := Stringify(query); actual != "SELECT id FROM users" {
		t.Fatalf("changed default rendering: %s", actual)
	}
	if query.Window == nil || query.Window.Raw != "LIMIT" {
		t.Fatalf("first window marker changed: %+v", query.Window)
	}
}

func TestBoundPaginationPreservesOperands(t *testing.T) {
	for _, sql := range []string{
		"SELECT id FROM records LIMIT :count OFFSET :start",
		"SELECT id FROM records LIMIT $count OFFSET $start",
		"SELECT id FROM records LIMIT ${Input.Count} OFFSET ${Input.Start}",
		"SELECT id FROM records LIMIT ? OFFSET ?",
		"SELECT r.* FROM (SELECT id FROM records ORDER BY id LIMIT :count OFFSET :start) r",
	} {
		parsed, err := ParseQuery(sql, WithStructuralValidation())
		if err != nil {
			t.Fatal(err)
		}
		got := (Stringifier{PreserveWindow: true}).String(parsed)
		reparsed, err := ParseQuery(got, WithStructuralValidation())
		if err != nil {
			t.Fatal(err)
		}
		if again := (Stringifier{PreserveWindow: true}).String(reparsed); again != got {
			t.Fatalf("unstable pagination %q -> %q", got, again)
		}
		if sql == "SELECT id FROM records LIMIT :count OFFSET :start" {
			if parsed.Limit.Kind != "placeholder" || parsed.Limit.Value != ":count" || parsed.Offset.Value != ":start" {
				t.Fatalf("lost bound operands: %+v", parsed)
			}
		}
	}
	for _, sql := range []string{"SELECT id FROM records LIMIT :count LIMIT :other", "SELECT id FROM records OFFSET $start OFFSET $other", "SELECT id FROM records LIMIT $count.call()"} {
		if _, err := ParseQuery(sql, WithStructuralValidation()); err == nil {
			t.Fatalf("invalid pagination accepted: %s", sql)
		}
	}
}

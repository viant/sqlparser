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

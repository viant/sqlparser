package sqlparser

import "testing"

func TestWildcardExplicitNameAmbiguity(t *testing.T) {
	for _, test := range []struct{ name, sql, column, table string }{
		{"cross-table wildcard first", "SELECT u.*,o.id AS id FROM users u JOIN orders o ON o.user_id=u.id", "id", ""},
		{"cross-table explicit first", "SELECT o.id AS id,u.* FROM users u JOIN orders o ON o.user_id=u.id", "id", ""},
		{"same-table duplicate", "SELECT u.*,u.id AS id FROM users u", "id", ""},
		{"multiple wildcards", "SELECT u.*,o.*,o.id AS id FROM users u JOIN orders o ON o.user_id=u.id", "id", ""},
		{"explicit wildcard exclusion", "SELECT u.* EXCEPT(id),o.id AS id FROM users u JOIN orders o ON o.user_id=u.id", "id", "orders"},
		{"computed alias remains unknown", "SELECT u.*,1 AS id FROM users u", "id", ""},
		{"known subquery absence", "SELECT u.*,o.id AS id FROM (SELECT name FROM users) u JOIN orders o ON o.user_id=1", "id", "orders"},
		{"nested duplicate", "SELECT outerq.id FROM (SELECT u.*,o.id AS id FROM users u JOIN orders o ON o.user_id=u.id) outerq", "id", ""},
		{"unaffected wildcard lookup", "SELECT u.*,o.id AS order_id FROM users u JOIN orders o ON o.user_id=u.id", "id", "users"},
		{"explicit without wildcard", "SELECT u.id AS user_id,o.id AS order_id FROM users u JOIN orders o ON o.user_id=u.id", "order_id", "orders"},
	} {
		t.Run(test.name, func(t *testing.T) {
			q, err := ParseQuery(test.sql)
			if err != nil {
				t.Fatal(err)
			}
			lineage := Lineage{Query: q}
			projection := lineage.Compile()
			origin, ok := projection.Lookup(test.column)
			if test.table == "" {
				if ok {
					t.Fatalf("ambiguous source attributed: %v", origin)
				}
				if _, ok := lineage.Columns()[test.column]; ok {
					t.Fatal("Columns retained ambiguous attribution")
				}
			} else if !ok || origin.Table != test.table {
				t.Fatalf("origin=%v ok=%v", origin, ok)
			}
		})
	}
}

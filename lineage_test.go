package sqlparser

import (
	"reflect"
	"testing"
)

func TestDirectColumnLineage(t *testing.T) {
	for _, test := range []struct {
		sql  string
		want map[string]ColumnOrigin
	}{
		{"SELECT u.id AS user_id, o.id AS order_id FROM users u JOIN orders o ON o.user_id=u.id", map[string]ColumnOrigin{"user_id": {"users", "id"}, "order_id": {"orders", "id"}}},
		{"SELECT id FROM users", map[string]ColumnOrigin{"id": {"users", "id"}}},
		{"SELECT id, u.id+1 AS computed FROM users u JOIN orders o ON o.user_id=u.id", map[string]ColumnOrigin{}},
		{"SELECT u.id AS id, o.id AS id FROM users u JOIN orders o ON o.user_id=u.id", map[string]ColumnOrigin{}},
	} {
		q, err := ParseQuery(test.sql)
		if err != nil {
			t.Fatal(err)
		}
		got := (Lineage{Query: q}).Columns()
		if !reflect.DeepEqual(test.want, got) {
			t.Fatalf("%s: got=%v want=%v", test.sql, got, test.want)
		}
	}
}

func TestWildcardLineageDoesNotAnnotateComputedOrAmbiguousColumns(t *testing.T) {
	for _, test := range []struct{ sql, column, table string }{
		{"SELECT * FROM users", "id", "users"}, {"SELECT *,1 AS id FROM users", "id", ""},
		{"SELECT u.*,o.id AS order_id FROM users u JOIN orders o ON o.user_id=u.id", "id", "users"},
		{"SELECT u.*,o.* FROM users u JOIN orders o ON o.user_id=u.id", "id", ""},
		{"SELECT u.id FROM users u JOIN orders u ON u.user_id=u.id", "id", ""},
	} {
		q, err := ParseQuery(test.sql)
		if err != nil {
			t.Fatal(err)
		}
		projection := (Lineage{Query: q}).Compile()
		origin, _ := projection.Lookup(test.column)
		if origin.Table != test.table {
			t.Fatalf("%s: %v", test.sql, origin)
		}
	}
}

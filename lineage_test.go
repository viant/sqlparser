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

func TestCTELineage(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want map[string]ColumnOrigin
	}{
		{
			name: "inherited CTE and nested shadow",
			sql: `WITH source AS (SELECT o.id FROM original o)
SELECT derived.inherited_id, derived.shadowed_id
FROM (
  WITH inner_source AS (SELECT source.id FROM source)
  SELECT inner_source.id AS inherited_id, nested.id AS shadowed_id
  FROM inner_source
  JOIN (WITH source AS (SELECT r.id FROM replacement r) SELECT source.id FROM source) nested
    ON nested.id = inner_source.id
) derived`,
			want: map[string]ColumnOrigin{"inherited_id": {"original", "id"}, "shadowed_id": {"replacement", "id"}},
		},
		{
			name: "recursive qualifier leaves independent key direct",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id, t.parent_id FROM taxonomy t
  UNION ALL
  SELECT child.id, child.parent_id FROM taxonomy child JOIN walk ON child.parent_id = walk.id
), level AS (
  SELECT top.id, walk.id AS qualifier_id, top.id + 1 AS computed FROM taxonomy top
  JOIN walk ON walk.id = top.id
)
SELECT level.id, level.qualifier_id, level.computed, level.missing FROM level`,
			want: map[string]ColumnOrigin{"id": {"taxonomy", "id"}},
		},
		{
			name: "recursive result remains opaque across inherited scope",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id FROM taxonomy t
  UNION ALL
  SELECT child.id FROM taxonomy child JOIN walk ON child.parent_id = walk.id
), derived AS (SELECT walk.id FROM walk)
SELECT derived.id FROM derived`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive self reference without union stays opaque",
			sql:  "WITH RECURSIVE walk AS (SELECT walk.id FROM walk) SELECT walk.id FROM walk",
			want: map[string]ColumnOrigin{},
		},
		{
			name: "mutually recursive joins cannot prove a physical key in either definition",
			sql: `WITH RECURSIVE a AS (
  SELECT t.id FROM taxonomy t JOIN b ON b.id = t.id
), b AS (
  SELECT t.id FROM taxonomy t JOIN a ON a.id = t.id
)
SELECT a.id, b.id AS qualifier_id FROM a JOIN b ON b.id = a.id`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "multihop recursive definitions remain opaque",
			sql: `WITH RECURSIVE a AS (
  SELECT t.id FROM taxonomy t JOIN b ON b.id = t.id
), b AS (
  SELECT t.id FROM taxonomy t JOIN c ON c.id = t.id
), c AS (
  SELECT t.id FROM taxonomy t JOIN a ON a.id = t.id
)
SELECT a.id, b.id AS qualifier_id, c.id AS computed FROM a JOIN b ON b.id = a.id JOIN c ON c.id = a.id`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive cycle in EXISTS cannot prove key",
			sql: `WITH RECURSIVE a AS (
  SELECT t.id FROM taxonomy t WHERE EXISTS (SELECT 1 FROM b WHERE b.id = t.id)
), b AS (
  SELECT a.id FROM a
)
SELECT b.id FROM b`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive union with unresolved mutual reference taints dependent definition",
			sql: `WITH RECURSIVE a AS (
  SELECT t.id FROM taxonomy t
  UNION ALL
  SELECT child.id FROM taxonomy child JOIN b ON b.id = child.id
), b AS (
  SELECT t.id FROM taxonomy t JOIN a ON a.id = t.id
)
SELECT b.id FROM b`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "self reference also blocks a directly projected key",
			sql:  "WITH RECURSIVE walk AS (SELECT t.id FROM taxonomy t JOIN walk ON walk.id = t.id) SELECT walk.id FROM walk",
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive reference in EXISTS makes definition opaque",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id FROM taxonomy t WHERE EXISTS (SELECT 1 FROM walk WHERE walk.id = t.id)
) SELECT walk.id FROM walk`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive reference in scalar expression makes definition opaque",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id, (SELECT MAX(walk.id) FROM walk) AS qualifier_id FROM taxonomy t
) SELECT walk.id FROM walk`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "nested CTE shadows recursive name inside EXISTS",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id FROM taxonomy t WHERE EXISTS (
    WITH walk AS (SELECT d.id FROM details d) SELECT 1 FROM walk WHERE walk.id = t.id
  )
) SELECT walk.id FROM walk`,
			want: map[string]ColumnOrigin{"id": {"taxonomy", "id"}},
		},
		{
			name: "unresolved forward declaration stays opaque",
			sql:  "WITH first AS (SELECT later.id FROM later), later AS (SELECT t.id FROM taxonomy t) SELECT first.id FROM first",
			want: map[string]ColumnOrigin{},
		},
		{
			name: "nonrecursive definition sees the inherited name it shadows",
			sql: `WITH source AS (SELECT o.id FROM original o)
SELECT nested.id FROM (
  WITH source AS (SELECT source.id FROM source) SELECT source.id FROM source
) nested`,
			want: map[string]ColumnOrigin{"id": {"original", "id"}},
		},
		{
			name: "recursive forward declaration shadows inherited name",
			sql: `WITH source AS (SELECT o.id FROM original o)
SELECT nested.id FROM (
  WITH RECURSIVE earlier AS (SELECT source.id FROM source),
  source AS (SELECT r.id FROM replacement r) SELECT earlier.id FROM earlier
) nested`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "recursive forward local shadow does not reference outer recursion",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id FROM taxonomy t WHERE EXISTS (
    WITH RECURSIVE first AS (SELECT id FROM walk),
    walk AS (SELECT id FROM details) SELECT id FROM first
  )
) SELECT walk.id FROM walk`,
			want: map[string]ColumnOrigin{"id": {"taxonomy", "id"}},
		},
		{
			name: "duplicate CTE declarations stay opaque",
			sql:  "WITH source AS (SELECT a.id FROM a), source AS (SELECT b.id FROM b) SELECT source.id FROM source",
			want: map[string]ColumnOrigin{},
		},
		{
			name: "opaque wildcard can overlap a direct key",
			sql: `WITH RECURSIVE walk AS (
  SELECT t.id FROM taxonomy t
  UNION ALL
  SELECT child.id FROM taxonomy child JOIN walk ON child.parent_id = walk.id
)
SELECT walk.*, t.id FROM taxonomy t JOIN walk ON walk.id = t.id`,
			want: map[string]ColumnOrigin{},
		},
		{
			name: "duplicate output origins remain ambiguous",
			sql: `WITH q AS (SELECT a.id AS first, b.id AS second FROM a JOIN b ON a.id = b.id)
SELECT q.first AS id, q.second AS id FROM q`,
			want: map[string]ColumnOrigin{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := ParseQuery(test.sql)
			if err != nil {
				t.Fatal(err)
			}
			before, err := ParseQuery(test.sql)
			if err != nil {
				t.Fatal(err)
			}
			projection := (Lineage{Query: parsed}).Compile()
			for _, column := range []string{"id", "inherited_id", "shadowed_id", "qualifier_id", "computed", "missing"} {
				got, ok := projection.Lookup(column)
				want, found := test.want[column]
				if got != want || ok != found {
					t.Errorf("Lookup(%q) = %v, %v, want %v, %v", column, got, ok, want, found)
				}
			}
			if !reflect.DeepEqual(parsed, before) {
				t.Fatalf("lineage mutated AST:\nbefore: %s\nafter: %s", Stringify(before), Stringify(parsed))
			}
		})
	}
}

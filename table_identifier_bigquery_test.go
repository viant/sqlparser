package sqlparser

import (
	"reflect"
	"strings"
	"testing"
)

func TestBigQueryTableIdentifierParts(t *testing.T) {
	for _, source := range []string{"`my-project.dataset.table`", "[my-project.dataset.table]", "[my-project:dataset.table]", "my-project.dataset.table", "my-project:dataset.table"} {
		got, err := (TableIdentifierParser{Product: "BigQuery"}).Parts(source)
		if err != nil || !reflect.DeepEqual(got, []string{"my-project", "dataset", "table"}) {
			t.Fatalf("%s: %v %v", source, got, err)
		}
	}
	for _, source := range []string{"`project.dataset.table`", "[project.dataset.table]", "[project:dataset.table]"} {
		q, err := ParseQuery("SELECT id FROM " + source)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(Stringify(q), "FROM "+source) {
			t.Fatalf("SQL spelling changed: %s", Stringify(q))
		}
		raw, _, err := SourceTable(q.From.X)
		if err != nil || raw != source {
			t.Fatalf("raw table=%s %v", raw, err)
		}
	}
	for _, product := range []string{"", "SQLServer", "SQLite", "MySQL"} {
		got, err := (TableIdentifierParser{Product: product}).Parts("[project.dataset.table]")
		if err != nil || !reflect.DeepEqual(got, []string{"project.dataset.table"}) {
			t.Fatalf("%s: %v %v", product, got, err)
		}
	}
	got, err := TableIdentifierParts(`"schema"."odd.table"`)
	if err != nil || !reflect.DeepEqual(got, []string{"schema", "odd.table"}) {
		t.Fatalf("ordinary identifier split: %v %v", got, err)
	}
	for _, source := range []string{"`p..t`", "[p:d:t]", "[p.d.t.extra]", "`p:d.t`"} {
		if _, err := (TableIdentifierParser{Product: "BigQuery"}).Parts(source); err == nil {
			t.Fatalf("invalid path accepted: %s", source)
		}
	}
}

package sqlparser

import (
	"reflect"
	"testing"
)

func TestBigQueryWildcardTableIdentifierParts(t *testing.T) {
	parser := TableIdentifierParser{Product: "BigQuery"}
	for _, source := range []string{
		"my-project.adrtbbid.ad_rtb_bidrequests_*",
		"`my-project.adrtbbid.ad_rtb_bidrequests_*`",
		"[my-project:adrtbbid.ad_rtb_bidrequests_*]",
		"my-project.adrtbbid.`ad_rtb_bidrequests_*`",
	} {
		got, err := parser.Parts(source)
		want := []string{"my-project", "adrtbbid", "ad_rtb_bidrequests_*"}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got=%v err=%v want=%v", source, got, err, want)
		}
	}
	for _, source := range []string{"events_*", "dataset.*", "project.dataset.*"} {
		if _, err := parser.Parts(source); err != nil {
			t.Fatalf("valid wildcard %s: %v", source, err)
		}
	}
	for _, source := range []string{
		"project*.dataset.table", "project.dataset*.table", "project.dataset.table**",
		"project.dataset.table*tail", "project.dataset.table* alias", "project.dataset.table*; SELECT 1",
		"`project*.dataset.table`", "`project.dataset.table**`", "`project.dataset.table*tail`",
	} {
		if _, err := parser.Parts(source); err == nil {
			t.Fatalf("invalid wildcard accepted: %s", source)
		}
	}
	for _, product := range []string{"", "MySQL", "SQLite", "SQLServer"} {
		if _, err := (TableIdentifierParser{Product: product}).Parts("events_*"); err == nil {
			t.Fatalf("%s accepted bare wildcard", product)
		}
	}
	query := "SELECT COUNT(*) FROM `my-project.adrtbbid.ad_rtb_bidrequests_*` WHERE _TABLE_SUFFIX = '20261011'"
	parsed, err := ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := SourceTable(parsed.From.X)
	if err != nil || raw != "`my-project.adrtbbid.ad_rtb_bidrequests_*`" {
		t.Fatalf("authored source changed: %s %v", raw, err)
	}
}

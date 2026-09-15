package sqlparser

import (
	"strings"
	"testing"
)

func TestBracedTableRootRoundTrip(t *testing.T) {
	for _, sql := range []string{
		"SELECT id FROM ${project}.ds.records",
		"SELECT id FROM $project.ds.records",
		"SELECT id FROM `${project}.ds.records`",
		"SELECT id FROM [${project}.ds.records]",
		"WITH rows AS (SELECT id FROM ${project}.ds.records) SELECT id FROM rows",
		"SELECT r.id FROM (SELECT id FROM ${project}.ds.records) r",
		"SELECT r.id FROM records r JOIN ${project}.ds.items i ON i.id = r.id",
	} {
		t.Run(sql, func(t *testing.T) {
			q, err := ParseQuery(sql)
			if err != nil {
				t.Fatal(err)
			}
			rendered := Stringify(q)
			marker := "${project}"
			if !strings.Contains(sql, marker) {
				marker = "$project"
			}
			if !strings.Contains(rendered, marker+".ds.") {
				t.Fatalf("lost authored selector: %s", rendered)
			}
			if _, err = ParseQuery(rendered); err != nil {
				t.Fatalf("roundtrip: %v", err)
			}
		})
	}
	q, err := ParseQuery("SELECT id FROM ${project}.ds.records")
	if err != nil {
		t.Fatal(err)
	}
	name, _, err := SourceTable(q.From.X)
	if err != nil || name != "${project}.ds.records" {
		t.Fatalf("source=%q error=%v", name, err)
	}
	if _, err = TableIdentifierParts(name); err == nil {
		t.Fatal("unresolved root became a physical identifier")
	}
}

func TestBracedTableRootRejectsMalformed(t *testing.T) {
	for _, source := range []string{"${}", "${project", "${project.ds}.records", "${project}suffix.ds.records", "${project}.ds.", "${project}..records", "${project}.ds..records"} {
		if _, err := ParseQuery("SELECT id FROM " + source); err == nil {
			t.Errorf("accepted malformed table %q", source)
		}
	}
}

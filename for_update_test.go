package sqlparser

import (
	"strings"
	"testing"
)

func TestForUpdateRoundTrip(t *testing.T) {
	type useCase struct{ description, input string }
	for _, tc := range []useCase{
		{"physical rows", "SELECT a.id FROM advertisers a WHERE a.id = ? FOR UPDATE"},
		{"template predicate boundary", "SELECT a.id FROM advertisers a WHERE a.id = ? $criteria FOR UPDATE"},
		{"no predicate", "SELECT id FROM advertisers FOR UPDATE"},
		{"derived source", "SELECT policy.* FROM (SELECT a.id FROM advertisers a WHERE a.id = ? FOR UPDATE) policy"},
		{"ordered limit", "SELECT id FROM advertisers ORDER BY id LIMIT 1 FOR UPDATE"},
	} {
		t.Run(tc.description, func(t *testing.T) {
			parsed, err := ParseQuery(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			rendered := (Stringifier{PreserveWindow: true}).String(parsed)
			if strings.Count(strings.ToUpper(rendered), "FOR UPDATE") != 1 {
				t.Fatalf("lost or duplicated row lock: %s", rendered)
			}
			if _, err = ParseQuery(rendered); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestForUpdateRejectsUnsupportedSuffix(t *testing.T) {
	for _, input := range []string{
		"SELECT id FROM advertisers WHERE id = FOR UPDATE",
		"SELECT id FROM advertisers FOR UPDATEX",
		"SELECT id FROM advertisers FOR UPDATE NOWAIT",
		"SELECT id FROM advertisers FOR UPDATE SKIP LOCKED",
		"SELECT id FROM advertisers FOR UPDATE OF advertisers",
		"SELECT id FROM advertisers FOR UPDATE FOR UPDATE",
		"SELECT id FROM advertisers FOR UPDATE WHERE id = 1",
		"SELECT id FROM advertisers FOR UPDATE; DELETE FROM advertisers",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseQuery(input); err == nil {
				t.Fatal("unsupported lock suffix accepted")
			}
		})
	}
}

func TestForUpdateDefaultStringifyPreservesInnerLock(t *testing.T) {
	parsed, err := ParseQuery("SELECT policy.* FROM (SELECT a.id FROM advertisers a WHERE a.id = ? FOR UPDATE) policy")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ForUpdate || parsed.NestedSelect() == nil || !parsed.NestedSelect().ForUpdate {
		t.Fatal("lock attached to wrong SELECT")
	}
	rendered := Stringify(parsed)
	reparsed, err := ParseQuery(rendered)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.ForUpdate || reparsed.NestedSelect() == nil || !reparsed.NestedSelect().ForUpdate {
		t.Fatalf("inner lock lost: %s", rendered)
	}
}

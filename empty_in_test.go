package sqlparser

import (
	"strings"
	"testing"
)

func TestEmptyInSet(t *testing.T) {
	for _, sql := range []string{
		`SELECT id FROM records WHERE id IN ()`,
		`SELECT id FROM records WHERE id NOT IN ( ) AND active = 1 ORDER BY id`,
		`SELECT id FROM records WHERE active = 1 OR id IN ()`,
		`WITH r AS (SELECT id FROM records WHERE id IN ()) SELECT id FROM r`,
		`SELECT r.id FROM (SELECT id FROM records WHERE id IN ()) r`,
	} {
		t.Run(sql, func(t *testing.T) {
			parsed, err := ParseQuery(sql)
			if err != nil {
				t.Fatal(err)
			}
			output := Stringify(parsed)
			if !strings.Contains(output, "IN (") {
				t.Fatalf("lost empty set: %s", output)
			}
			if strings.Contains(sql, "AND active") && (!strings.Contains(output, "active") || len(parsed.OrderBy) != 1) {
				t.Fatalf("lost following predicate/order: %s", output)
			}
		})
	}
	for _, sql := range []string{`SELECT () FROM records`, `SELECT f(()) FROM records`, `SELECT id FROM records WHERE id IN (,)`, `SELECT id FROM records WHERE id IN (1,)`} {
		if _, err := ParseQuery(sql); err == nil {
			t.Fatalf("accepted malformed SQL: %s", sql)
		}
	}
}

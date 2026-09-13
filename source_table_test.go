package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"reflect"
	"testing"
)

func TestSourceTableClassifiesFullSource(t *testing.T) {
	for _, test := range []struct {
		raw, want string
		wrapped   bool
	}{{"records", "records", false}, {"(records)", "records", true}, {"(schema.records)", "schema.records", true}, {"((records))", "records", true}, {"(selection_log)", "selection_log", true}, {"(SELECT * FROM records)", "", false}, {"(records) alias", "", false}, {"(records OR 1=1)", "", false}, {"(records,other)", "", false}, {"(records JOIN other ON 1=1)", "", false},
		{"($View.Users.NonWindowSQL)", "", false}, {"(${View.Users.NonWindowSQL})", "", false}, {"(${embed:queries:rows.sql})", "", false}, {"(:table)", "", false}, {"($)", "", false}, {"(`$literal`)", "`$literal`", true}, {`("$literal")`, `"$literal"`, true}, {"([$literal])", "[$literal]", true}, {"(price$history)", "price$history", true},
	} {
		t.Run(test.raw, func(t *testing.T) {
			source := &expr.Raw{Raw: test.raw}
			before := *source
			name, wrapped, err := SourceTable(source)
			if err != nil || name != test.want || wrapped != test.wrapped {
				t.Fatalf("table=%q wrapped=%v err=%v", name, wrapped, err)
			}
			if !reflect.DeepEqual(before, *source) {
				t.Fatal("classifier mutated AST")
			}
		})
	}
}

func TestSourceTableClassifiesParsedFromAndJoin(t *testing.T) {
	query, err := ParseQuery("SELECT r.*, a.* FROM (records) r JOIN (auxiliary) a ON a.id=r.id")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []struct {
		name  string
		value any
	}{{"records", query.From.X}, {"auxiliary", query.Joins[0].With}} {
		name, wrapped, err := SourceTable(source.value)
		if err != nil || name != source.name || !wrapped {
			t.Fatalf("table=%s wrapped=%v err=%v", name, wrapped, err)
		}
	}
}

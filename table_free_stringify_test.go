package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"testing"
)

func TestStringifyTableFreeSelect(t *testing.T) {
	for _, source := range []string{"SELECT 1 AS id", "SELECT 1 AS id, 'a' AS name", "SELECT id FROM records"} {
		parsed, err := ParseQuery(source)
		if err != nil {
			t.Fatal(err)
		}
		if got := Stringify(parsed); got != source {
			t.Fatalf("rendered %q, want %q", got, source)
		}
		parsed.Qualify = &expr.Qualify{X: &expr.Binary{X: expr.NewIntLiteral("1"), Op: "=", Y: expr.NewIntLiteral("0")}}
		want := source + " WHERE 1 = 0"
		if got := Stringify(parsed); got != want {
			t.Fatalf("qualified %q, want %q", got, want)
		}
	}
}

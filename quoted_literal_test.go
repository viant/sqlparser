package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"testing"
)

func TestQuotedLiteralEscapes(t *testing.T) {
	for _, source := range []string{"'O''Brien'", "'O\\'Brien'", `"a""b"`} {
		t.Run(source, func(t *testing.T) {
			cursor := parsly.NewCursor("", []byte("name = "+source+" "), 0)
			qualify := &expr.Qualify{}
			if err := ParseQualify(cursor, qualify); err != nil {
				t.Fatal(err)
			}
			binary, ok := qualify.X.(*expr.Binary)
			if !ok {
				t.Fatalf("unexpected %T", qualify.X)
			}
			literal, ok := binary.Y.(*expr.Literal)
			if !ok || literal.Value != source {
				t.Fatalf("literal %#v", binary.Y)
			}
			if cursor.Pos < len(cursor.Input)-1 {
				t.Fatalf("unconsumed input %q", cursor.Input[cursor.Pos:])
			}
		})
	}
}

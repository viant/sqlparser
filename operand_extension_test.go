package sqlparser

import (
	"strings"
	"testing"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestOperandExtension(t *testing.T) {
	extension := WithErrorHandler(func(err error, cursor *parsly.Cursor, destination any) error {
		operand, ok := destination.(*node.Node)
		if !ok || !strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "@values") {
			return err
		}
		cursor.Pos += len("@values")
		*operand = expr.NewRaw("@values")
		return nil
	})
	for _, sql := range []string{
		`SELECT id FROM records WHERE id IN (@values) ORDER BY id`,
		`SELECT COALESCE(@values, 1) AS id FROM records`,
		`SELECT id FROM records WHERE id IN (0, @values, 2)`,
		`WITH r AS (SELECT id FROM records WHERE id IN (@values)) SELECT id FROM r`,
		`SELECT r.id FROM (SELECT id FROM records WHERE id IN (@values)) r`,
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := ParseQuery(sql); err == nil {
				t.Fatal("extension must be opt-in")
			}
			parsed, err := ParseQuery(sql, extension)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(Stringify(parsed), "@values") {
				t.Fatalf("lost raw operand: %s", Stringify(parsed))
			}
		})
	}
	for _, sql := range []string{
		`SELECT f(@values,) FROM records`, `SELECT f(@unknown) FROM records`,
		`SELECT f(1 +) FROM records`, `SELECT f(@values 2) FROM records`,
	} {
		if _, err := ParseQuery(sql, extension); err == nil {
			t.Fatalf("accepted malformed SQL: %s", sql)
		}
	}
}

func TestOperandExtensionRequiresProgressAndNode(t *testing.T) {
	for _, mode := range []string{"no progress", "no node", "past end"} {
		t.Run(mode, func(t *testing.T) {
			_, err := ParseQuery(`SELECT f(@value) FROM records`, WithErrorHandler(func(err error, cursor *parsly.Cursor, destination any) error {
				operand, ok := destination.(*node.Node)
				if !ok {
					return err
				}
				if mode != "no node" {
					*operand = expr.NewRaw("@value")
				}
				if mode != "no progress" {
					cursor.Pos = len(cursor.Input)
				}
				if mode == "past end" {
					cursor.Pos++
				}
				return nil
			}))
			if err == nil {
				t.Fatal("invalid extension accepted")
			}
		})
	}
}

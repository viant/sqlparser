package sqlparser

import (
	"strings"
	"testing"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/query"
)

func TestPlaceholderBeforeOrderBy(t *testing.T) {
	sql := `SELECT id FROM records WHERE 1=1 ${predicate.Builder().Build("AND")} ORDER BY id`
	parsed, err := ParseQuery(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.OrderBy) != 1 || !strings.Contains(Stringify(parsed), `${predicate.Builder().Build("AND")}`) {
		t.Fatalf("lost template/order: %s", Stringify(parsed))
	}
}

func TestErrorHandlerResumesFollowingClause(t *testing.T) {
	parsed, err := ParseQuery("SELECT id FROM records @suffix ORDER BY id", WithErrorHandler(func(err error, cursor *parsly.Cursor, node any) error {
		from, ok := node.(*query.From)
		if !ok || !strings.HasPrefix(string(cursor.Input[cursor.Pos:]), "@suffix") {
			return err
		}
		from.Unparsed = "@suffix"
		cursor.Pos += len("@suffix")
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.OrderBy) != 1 || parsed.From.Unparsed != "@suffix" {
		t.Fatalf("recovery lost clause: %+v", parsed)
	}
}

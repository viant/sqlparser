package sqlparser

import (
	"reflect"
	"testing"
)

func TestTableIdentifierParts(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []string
	}{
		{"records", []string{"records"}}, {"RECORDS", []string{"RECORDS"}},
		{"main.records", []string{"main", "records"}}, {" main . `records` ", []string{"main", "records"}},
		{`"main"."odd.name"`, []string{"main", "odd.name"}},
		{"[main].[odd name]", []string{"main", "odd name"}},
		{`"a""b"`, []string{`a"b`}}, {"`a``b`", []string{"a`b"}},
		{"'a''b'", []string{"a'b"}}, {"[a]]b]", []string{"a]b"}},
		{"κατάλογος.πίνακας", []string{"κατάλογος", "πίνακας"}},
	} {
		t.Run(tc.source, func(t *testing.T) {
			actual, err := TableIdentifierParts(tc.source)
			if err != nil || !reflect.DeepEqual(actual, tc.want) {
				t.Fatalf("parts=%v err=%v want=%v", actual, err, tc.want)
			}
		})
	}
	for _, source := range []string{"", "main.", ".records", "records alias", "records; DROP TABLE x", "(records)", "$TABLE", "records()", "records/*x*/", "\"unterminated", "[]", "a..b", "a/b"} {
		if _, err := TableIdentifierParts(source); err == nil {
			t.Fatalf("accepted invalid reference %q", source)
		}
	}
}

package sqlparser_test

import (
	"github.com/viant/sqlparser"
	"reflect"
	"testing"
)

func TestParseInsertQuotedTableTarget(t *testing.T) {
	for _, tc := range []struct {
		name, sql, target string
		columns           []string
	}{
		{"double quoted SQLX", `INSERT INTO "session/attributes"("value","user_id","name") VALUES (?,?,?)`, `"session/attributes"`, []string{`"value"`, `"user_id"`, `"name"`}},
		{"backtick", "INSERT INTO `session/attributes`(`value`,`user_id`,`name`) VALUES (?,?,?)", "`session/attributes`", []string{"`value`", "`user_id`", "`name`"}},
		{"unquoted", "INSERT INTO session(value,user_id,name) VALUES (?,?,?)", "session", []string{"value", "user_id", "name"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sqlparser.ParseInsert(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			if got.Target.X == nil {
				t.Fatal("successful INSERT parse has nil target")
			}
			if target := sqlparser.Stringify(got.Target.X); target != tc.target {
				t.Fatalf("target=%s", target)
			}
			if !reflect.DeepEqual(got.Columns, tc.columns) || len(got.Values) != 3 {
				t.Fatalf("columns/values changed: %+v", got)
			}
			rendered := sqlparser.Stringify(got)
			again, err := sqlparser.ParseInsert(rendered)
			if err != nil || again.Target.X == nil || !reflect.DeepEqual(again.Columns, tc.columns) {
				t.Fatalf("round trip failed: %s error=%v", rendered, err)
			}
		})
	}
}
func TestParseInsertRejectsMissingOrMalformedTarget(t *testing.T) {
	for _, source := range []string{`INSERT INTO`, `INSERT INTO (value) VALUES (?)`, `INSERT INTO "unterminated(value) VALUES (?)`, "INSERT INTO `unterminated(value) VALUES (?)", `INSERT INTO ?(value) VALUES (?)`} {
		if _, err := sqlparser.ParseInsert(source); err == nil {
			t.Fatalf("malformed target accepted: %s", source)
		}
	}
}

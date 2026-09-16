package sqlparser

import (
	"testing"

	"github.com/viant/sqlparser/query"
)

func TestExistsSubqueryArguments(t *testing.T) {
	for _, expression := range []string{
		"EXISTS (SELECT 1)",
		"EXISTS ((SELECT 1 FROM targeting))",
		"EXISTS (((SELECT 1 FROM targeting)))",
		"EXISTS ( /* outer */ ( /* inner */ SELECT 1 FROM targeting ) /* trailing */ )",
		"EXISTS (SELECT 1 FROM targeting tr WHERE tr.id = ao.id AND tr.value IN (?))",
		"exists (\n SELECT 1\n FROM targeting tr\n WHERE tr.id = ao.id\n)",
		"EXISTS (WITH target AS (SELECT id FROM targeting) SELECT 1 FROM target WHERE id = ao.id)",
		"EXISTS ((WITH target AS (SELECT id FROM targeting) SELECT 1 FROM target WHERE id = ao.id))",
		"EXISTS ((SELECT id FROM targeting UNION ALL SELECT id FROM targeting))",
	} {
		t.Run(expression, func(t *testing.T) {
			call, err := ParseCallExpr(expression)
			if err != nil {
				t.Fatal(err)
			}
			if len(call.Args) != 1 {
				t.Fatalf("arguments: %#v", call.Args)
			}
			if _, ok := call.Args[0].(*query.Select); !ok {
				t.Fatalf("argument type: %T", call.Args[0])
			}
			for _, predicate := range []string{expression, "NOT (" + expression + ")", "(" + expression + ") AND ao.id IN (?)"} {
				sql := "SELECT ao.id, SUM(ao.bids) AS bids FROM perf ao WHERE " + predicate + " GROUP BY ao.id HAVING bids > ?"
				parsed, err := ParseQuery(sql)
				if err != nil {
					t.Fatal(err)
				}
				rendered := Stringify(parsed)
				again, err := ParseQuery(rendered)
				if err != nil {
					t.Fatalf("round trip: %s: %v", rendered, err)
				}
				if Stringify(again) != rendered {
					t.Fatalf("unstable serialization: %s", rendered)
				}
				if len(again.GroupBy) != 1 || again.Having == nil {
					t.Fatal("grouping/having lost")
				}
			}
		})
	}
}

func TestExistsRejectsScalarArguments(t *testing.T) {
	for _, expression := range []string{"EXISTS()", "EXISTS(1)", "EXISTS(a,b)", "EXISTS((1))", "EXISTS(())", "EXISTS((SELECT 1 FROM t), 2)", "EXISTS((SELECT 1 FROM t) + 1)", "EXISTS(SELECT 1 FROM t), extra"} {
		if _, err := ParseCallExpr(expression); err == nil {
			t.Fatalf("accepted %s", expression)
		}
	}
}

func TestExistsRejectsIncompleteQueries(t *testing.T) {
	for _, body := range []string{
		"SELECT", "SELECT FROM t", "SELECT 1 +", "WITH c AS (SELECT 1)",
		"WITH c AS (SELECT FROM t) SELECT 1 FROM c",
		"SELECT 1 FROM t UNION ALL SELECT FROM t",
	} {
		for _, enclosed := range []string{body, "(" + body + ")", "((" + body + "))"} {
			expression := "EXISTS(" + enclosed + ")"
			t.Run(expression, func(t *testing.T) {
				if _, err := ParseCallExpr(expression); err == nil {
					t.Fatal("accepted incomplete EXISTS query")
				}
				SQL := "SELECT id FROM src WHERE " + expression
				if _, err := ParseQuery(SQL); err == nil {
					t.Fatal("accepted incomplete EXISTS query in WHERE")
				}
			})
		}
	}
}

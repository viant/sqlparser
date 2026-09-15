package sqlparser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestCallArgumentsStrict(t *testing.T) {
	for _, sql := range []string{
		"f(a,)", "f(,a)", "f(a,,b)", "f(a,'b' extra)", "f(a,'b' 'c')",
		"f(a + b extra)", "f(a +)", "f(a AND)", "f(g(a,))", "f((g(a,)))",
		"f((a extra))", "f(a AS b)", "f(a, /* missing */)", "f(a) extra",
		"f(a BETWEEN 1 AND)", "f(a BETWEEN AND 2)", "f(CASE WHEN a THEN g(b,) END)",
		"f(a ORDER BY b,)", "f(a ORDER BY b extra)", "f(a ORDER BY b LIMIT)",
		"f(DISTINCT)", "f(a #if($x) , hidden(b) #end)",
		"CAST(a AS INT,)", "CAST(a AS)", "CAST(a extra AS INT)",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := ParseCallExpr(sql); err == nil {
				t.Fatal("accepted malformed call")
			}
			if !strings.HasSuffix(sql, ") extra") {
				if _, err := ParseQuery("SELECT " + sql + " FROM t"); err == nil {
					t.Fatal("query accepted malformed call")
				}
			}
		})
	}
}

func TestCallArgumentsSource(t *testing.T) {
	for _, sql := range []string{
		"f()", "f(a, g(b, 'x,y'))", "f(a + b * c, NOT flag)",
		"CAST(coalesce(a, 0) AS DECIMAL(10,2))", "CAST(a AS DOUBLE PRECISION)",
		"CAST(a, 'app.Pair[A,B]')", "count(DISTINCT t.id)",
		"JSON_ARRAYAGG(name ORDER BY depth ASC)", "f($value, ${params.Value})",
		"ARRAY_AGG(name ORDER BY depth DESC, id ASC LIMIT 1)",
		"DATE_SUB(CURRENT_DATE(), INTERVAL $days DAY)", "f(INTERVAL '1 day')",
		"f((a,b), c IN (1,2))", "f((SELECT g(a) FROM t))",
		"f(a /* retained */, b)", "f('CASE WHEN invariant(x) END')",
	} {
		t.Run(sql, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + sql + " FROM t")
			if err != nil {
				t.Fatal(err)
			}
			if got := Stringify(q.List[0].Expr); got != sql {
				t.Fatalf("source changed: %q", got)
			}
			if _, err = ParseCallExpr(sql); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Follow node fields, never Raw text, just as AST consumers do.
func callNames(n node.Node) []string {
	var names []string
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return
			}
			if v.Kind() == reflect.Pointer && v.CanInterface() {
				if call, ok := v.Interface().(*expr.Call); ok {
					names = append(names, Stringify(call.X))
				}
			}
			if v.Kind() == reflect.Interface {
				visit(v.Elem())
				return
			}
			v = v.Elem()
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	visit(reflect.ValueOf(n))
	return names
}

func TestCaseStructured(t *testing.T) {
	for _, sql := range []string{
		"CASE selector(a) WHEN branch(b) THEN result(c) ELSE fallback(d) END",
		"CASE WHEN condition(a)=1 AND b=2 THEN result(c) ELSE fallback(d) END",
		"case selector(a) when 1 then case when condition(b)=2 then result(c) else fallback(d) end else final(e) end",
		"CASE WHEN a BETWEEN 1 AND 2 THEN result(c) WHEN b=2 THEN other(d) END",
		"CASE WHEN a=1 THEN CAST(result(c) AS INTEGER) ELSE fallback(d) END",
		"CASE WHEN weekend=1 /* END WHEN */ THEN result(c) ELSE 'END CASE WHEN' END",
	} {
		t.Run(sql, func(t *testing.T) {
			q, err := ParseQuery("SELECT " + sql + " AS value FROM t")
			if err != nil {
				t.Fatal(err)
			}
			sw, ok := q.List[0].Expr.(*expr.Switch)
			if !ok || len(sw.Cases) == 0 {
				t.Fatalf("missing CASE AST: %#v", q.List[0].Expr)
			}
			if got := Stringify(sw); got != sql {
				t.Fatalf("source changed: %q", got)
			}
			names := strings.Join(callNames(sw), ",")
			for _, name := range []string{"selector", "branch", "condition", "result", "fallback", "final", "other", "CAST"} {
				if strings.Contains(sql, name+"(") && !strings.Contains(names, name) {
					t.Fatalf("%s absent from AST: %s", name, names)
				}
			}
		})
	}
}

func TestCaseMalformed(t *testing.T) {
	for _, sql := range []string{
		"CASE END", "CASE a END", "CASE WHEN THEN 1 END", "CASE WHEN a THEN END",
		"CASE WHEN a THEN 1 ELSE END", "CASE WHEN a THEN f(x,) END",
		"CASE WHEN a extra THEN 1 END", "CASE WHEN a THEN 1 extra END",
		"CASE WHEN a THEN 1", "CASE WHEN a THEN 1 ELSE 2 ELSE 3 END",
		"CASE WHEN a THEN #if($x) f(a) #end END",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := ParseQuery("SELECT " + sql + " FROM t"); err == nil {
				t.Fatal("accepted incomplete or opaque CASE")
			}
		})
	}
}

func TestCaseStripCollate(t *testing.T) {
	for _, sql := range []string{
		"CASE selector(a COLLATE nocase) WHEN 'x' THEN result(b COLLATE nocase) ELSE fallback(c) END",
		"CASE WHEN a COLLATE nocase = 'x' THEN CASE b WHEN 1 THEN c COLLATE nocase ELSE d END ELSE e END",
	} {
		got, err := StripCollate("SELECT " + sql + " FROM t")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "COLLATE") {
			t.Fatalf("collation survived: %s", got)
		}
		if _, err = ParseQuery(got); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCallCasePublicLayout(t *testing.T) {
	_ = expr.Raw{"raw", nil, ""}
	_ = expr.Switch{"raw", expr.Ident{}, nil}
	_ = expr.Case{expr.Qualify{}, nil}
}

func TestCaseSelectorAndElseNodes(t *testing.T) {
	q, err := ParseQuery("SELECT CASE choose(a) WHEN 1 THEN yes(b) WHEN 2 THEN maybe(c) ELSE no(d) END FROM t")
	if err != nil {
		t.Fatal(err)
	}
	sw := q.List[0].Expr.(*expr.Switch)
	if len(sw.Cases) != 4 || sw.Cases[0].Y != nil || sw.Cases[3].X.X != nil {
		t.Fatalf("selector/ELSE layout: %#v", sw.Cases)
	}
	if got := callNames(sw); !reflect.DeepEqual(got, []string{"choose", "yes", "maybe", "no"}) {
		t.Fatalf("walked calls: %v", got)
	}
	sw.Raw = ""
	if got := Stringify(sw); got != "CASE choose(a) WHEN 1 THEN yes(b) WHEN 2 THEN maybe(c) ELSE no(d) END" {
		t.Fatalf("structured rendering: %s", got)
	}
}

func TestCallCaseTransformFidelity(t *testing.T) {
	for _, sql := range []string{
		"SELECT CASE WHEN ${predicate.Test} THEN $View.Value(a) ELSE 0 END FROM t",
		"SELECT CASE WHEN a=1 /* vendor hint */ THEN CAST(b AS DECIMAL(10,2)) ELSE 0 END FROM t",
		"SELECT CAST($value AS app.Pair[A,B]) FROM t",
		"SELECT f(a /* retained */, b) FROM t",
	} {
		got, err := StripCollate(sql)
		if err != nil {
			t.Fatal(err)
		}
		if got != sql {
			t.Fatalf("untransformed source changed:\n%s\n%s", sql, got)
		}
	}
	for _, sql := range []string{
		"SELECT CASE WHEN a = 'x' THEN CAST(f(b COLLATE nocase) AS DECIMAL(10,2)) ELSE 0 END FROM t",
		"SELECT ARRAY_AGG(f(a COLLATE nocase) ORDER BY g(b COLLATE nocase) DESC, c ASC LIMIT 1) FROM t",
		"SELECT f(INTERVAL g(a COLLATE nocase) DAY) FROM t",
		"SELECT CASE WHEN a IN ('x' COLLATE nocase, 'y') THEN f(b) ELSE g(c) END FROM t",
	} {
		q, err := ParseQuery(sql)
		if err != nil {
			t.Fatal(err)
		}
		before := callNames(q)
		got, err := StripCollate(sql)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "COLLATE") {
			t.Fatalf("collation survived: %s", got)
		}
		q, err = ParseQuery(got)
		if err != nil {
			t.Fatalf("%s: %v", got, err)
		}
		if after := callNames(q); !reflect.DeepEqual(before, after) {
			t.Fatalf("calls changed: %v -> %v (%s)", before, after, got)
		}
		for _, token := range []string{"AS", "ORDER BY", "DESC", "ASC", "LIMIT", "INTERVAL", "DAY"} {
			if strings.Contains(sql, token) && !strings.Contains(got, token) {
				t.Fatalf("%s lost: %s", token, got)
			}
		}
	}
}

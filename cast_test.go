package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"testing"
)

func TestCastExpressionRetainsAuthoredTypes(t *testing.T) {
	for _, tc := range []struct{ sql, operand, typ string }{
		{"CAST(r.pseudo AS app.Shape)", "r.pseudo", "app.Shape"},
		{"cast(r.pseudo AS '[]*app.Shape')", "r.pseudo", "[]*app.Shape"},
		{"cast(r.pseudo, 'app.Pair[A,B]')", "r.pseudo", "app.Pair[A,B]"},
		{"CAST(COALESCE(' AS ', r.value) AS DECIMAL(10,2))", "COALESCE(' AS ', r.value)", "DECIMAL(10,2)"},
		{"CAST(r.pseudo /* AS ignored */ AS app.Shape)", "r.pseudo /* AS ignored */", "app.Shape"},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			for _, query := range []bool{false, true} {
				var call *expr.Call
				var err error
				if query {
					parsed, e := ParseQuery("SELECT " + tc.sql + " FROM records r")
					err = e
					if e == nil {
						call = parsed.List[0].Expr.(*expr.Call)
					}
				} else {
					call, err = ParseCallExpr(tc.sql)
				}
				if err != nil {
					t.Fatal(err)
				}
				got, err := CastExpression(call)
				if err != nil || got.Operand != tc.operand || got.Type != tc.typ {
					t.Fatalf("got=%+v err=%v", got, err)
				}
			}
		})
	}
	for _, raw := range []string{"()", "(a)", "( AS int)", "(a AS )", "(a AS int) tail", "(a,int,extra)"} {
		if _, err := CastExpression(&expr.Call{X: expr.NewSelector("cast"), Raw: raw}); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

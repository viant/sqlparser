package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"testing"
)

func TestIntervalArithmeticUsesExistingExpressionParser(t *testing.T) {
	for _, amount := range []string{":seconds * 2", "15 * 2", "-15", "(:seconds + 1) * 2", ":seconds /* amount */ * 2", "1 + 2 * 3", "CAST(:seconds AS SIGNED) * 2"} {
		t.Run(amount, func(t *testing.T) {
			SQL := "SELECT CURRENT_TIMESTAMP() - INTERVAL " + amount + " SECOND AS cutoff FROM records WHERE id=1"
			parsed, err := ParseQuery(SQL)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Qualify == nil || len(parsed.List) != 1 || parsed.List[0].Alias != "cutoff" {
				t.Fatalf("interval consumed surrounding SQL: %#v", parsed)
			}
			arithmetic, ok := parsed.List[0].Expr.(*expr.Binary)
			if !ok {
				t.Fatalf("expected subtraction, got %T", parsed.List[0].Expr)
			}
			interval, ok := arithmetic.Y.(*expr.Unary)
			if !ok {
				t.Fatalf("expected interval operand, got %T", arithmetic.Y)
			}
			unit, ok := interval.X.(*expr.Binary)
			if !ok {
				t.Fatalf("expected interval unit, got %T", interval.X)
			}
			name, ok := unit.Y.(*expr.Ident)
			if !ok || name.Name != "SECOND" {
				t.Fatalf("unit lost: %#v", unit.Y)
			}
			if unit.X == nil {
				t.Fatal("amount lost")
			}
			if amount == "1 + 2 * 3" {
				addition, ok := unit.X.(*expr.Binary)
				if !ok || addition.Op != "+" {
					t.Fatalf("expected addition in amount, got %#v", unit.X)
				}
				product, ok := addition.Y.(*expr.Binary)
				if !ok || product.Op != "*" {
					t.Fatalf("multiplication precedence lost: %#v", addition.Y)
				}
			}
		})
	}
}

func TestIntervalArithmeticKeepsDerivedSourceAndClauseBoundaries(t *testing.T) {
	SQL := "SELECT journal.* FROM (SELECT COUNT(*) AS total FROM records WHERE created >= CURRENT_TIMESTAMP() - INTERVAL :seconds * 2 SECOND AND id=1) journal ORDER BY total DESC"
	parsed, err := ParseQuery(SQL)
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := parsed.From.X.(*expr.Raw)
	if !ok || raw.Raw != "(SELECT COUNT(*) AS total FROM records WHERE created >= CURRENT_TIMESTAMP() - INTERVAL :seconds * 2 SECOND AND id=1)" {
		t.Fatalf("derived SQL changed: %#v", parsed.From.X)
	}
	if parsed.From.Alias != "journal" || len(parsed.OrderBy) != 1 {
		t.Fatal("outer SQL boundary lost")
	}
}

func TestIntervalArithmeticRequiresCompleteAmountAndUnit(t *testing.T) {
	for _, SQL := range []string{"SELECT INTERVAL 1 * SECOND FROM records", "SELECT INTERVAL 1 + FROM records", "SELECT INTERVAL 1 * 2 BOGUS FROM records", "SELECT INTERVAL 1 * 2 FROM records"} {
		if _, err := ParseQuery(SQL); err == nil {
			t.Fatalf("invalid interval accepted: %s", SQL)
		}
	}
}

func TestIntervalQuotedLiteralKeepsFollowingArithmetic(t *testing.T) {
	parsed, err := ParseQuery("SELECT INTERVAL '1 day' + INTERVAL '2 hours' AS duration FROM records")
	if err != nil {
		t.Fatal(err)
	}
	addition, ok := parsed.List[0].Expr.(*expr.Binary)
	if !ok || addition.Op != "+" {
		t.Fatalf("expected addition between intervals, got %#v", parsed.List[0].Expr)
	}
	for _, operand := range []interface{}{addition.X, addition.Y} {
		interval, ok := operand.(*expr.Unary)
		if !ok {
			t.Fatalf("expected interval, got %T", operand)
		}
		if _, ok := interval.X.(*expr.Literal); !ok {
			t.Fatalf("quoted interval consumed following expression: %#v", interval.X)
		}
	}
}

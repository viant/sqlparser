package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestBinaryPrecedenceEqualityAndLogicalOperators(t *testing.T) {
	q, err := ParseQuery("SELECT * FROM t WHERE a = b AND c = d OR e = f")
	require.NoError(t, err)
	root := requireBinaryOp(t, q.Qualify.X, "OR")
	left := requireBinaryOp(t, root.X, "AND")
	assertBinarySQL(t, left.X, "a = b")
	assertBinarySQL(t, left.Y, "c = d")
	assertBinarySQL(t, root.Y, "e = f")

	q, err = ParseQuery("SELECT * FROM t WHERE a = b OR c = d AND e = f")
	require.NoError(t, err)
	root = requireBinaryOp(t, q.Qualify.X, "OR")
	assertBinarySQL(t, root.X, "a = b")
	right := requireBinaryOp(t, root.Y, "AND")
	assertBinarySQL(t, right.X, "c = d")
	assertBinarySQL(t, right.Y, "e = f")
}

func TestBinaryPrecedencePreservesExplicitParentheses(t *testing.T) {
	q, err := ParseQuery("SELECT * FROM t WHERE a = (b AND c = d)")
	require.NoError(t, err)
	root := requireBinaryOp(t, q.Qualify.X, "=")
	_, ok := root.Y.(*expr.Parenthesis)
	require.True(t, ok, "explicit right-hand parentheses must remain opaque")

	q, err = ParseQuery("SELECT * FROM t WHERE (a = b OR c = d) AND e = f")
	require.NoError(t, err)
	root = requireBinaryOp(t, q.Qualify.X, "AND")
	_, ok = root.X.(*expr.Parenthesis)
	require.True(t, ok, "explicit left-hand grouping must remain parenthesized")
	assertBinarySQL(t, root.Y, "e = f")
}

func TestBinaryPrecedenceArithmeticLeftAssociativity(t *testing.T) {
	for _, test := range []struct {
		name string
		sql  string
		op   string
		left string
		y    string
	}{
		{name: "subtraction", sql: "SELECT a - b - c FROM t", op: "-", left: "a - b", y: "c"},
		{name: "division", sql: "SELECT a / b / c FROM t", op: "/", left: "a / b", y: "c"},
		{name: "additive mixed", sql: "SELECT a - b + c FROM t", op: "+", left: "a - b", y: "c"},
		{name: "multiplicative mixed", sql: "SELECT a / b * c FROM t", op: "*", left: "a / b", y: "c"},
	} {
		t.Run(test.name, func(t *testing.T) {
			q, err := ParseQuery(test.sql)
			require.NoError(t, err)
			root := requireBinaryOp(t, q.List[0].Expr, test.op)
			assertBinarySQL(t, root.X, test.left)
			assert.Equal(t, test.y, Stringify(root.Y))
		})
	}
}

func TestBinaryPrecedencePreservesUnaryOperandHandling(t *testing.T) {
	q, err := ParseQuery("SELECT * FROM t WHERE NOT a = b AND c = d")
	require.NoError(t, err)
	root := requireBinaryOp(t, q.Qualify.X, "AND")
	left := requireBinaryOp(t, root.X, "=")
	_, ok := left.X.(*expr.Unary)
	require.True(t, ok, "existing unary operand shape changed")
	assertBinarySQL(t, root.Y, "c = d")
}

func TestBinaryPrecedenceJoinMultipleKeys(t *testing.T) {
	q, err := ParseQuery("SELECT * FROM left_table l JOIN right_table r ON l.id = r.id AND l.region = r.region AND l.kind = r.kind")
	require.NoError(t, err)
	require.Len(t, q.Joins, 1)
	terms := collectLogicalTerms(t, q.Joins[0].On.X, "AND")
	require.Len(t, terms, 3)
	assertBinarySQL(t, terms[0], "l.id = r.id")
	assertBinarySQL(t, terms[1], "l.region = r.region")
	assertBinarySQL(t, terms[2], "l.kind = r.kind")
}

func TestBinaryPrecedenceRejectsInvalidExpression(t *testing.T) {
	_, err := ParseQuery("SELECT * FROM t WHERE a = AND b = c")
	require.Error(t, err)
}

func requireBinaryOp(t *testing.T, n node.Node, op string) *expr.Binary {
	t.Helper()
	binary, ok := n.(*expr.Binary)
	require.True(t, ok, "expected *expr.Binary, got %T", n)
	require.Equal(t, op, strings.ToUpper(strings.TrimSpace(binary.Op)))
	return binary
}

func assertBinarySQL(t *testing.T, n node.Node, expected string) {
	t.Helper()
	_, ok := n.(*expr.Binary)
	require.True(t, ok, "expected *expr.Binary, got %T", n)
	assert.Equal(t, expected, Stringify(n))
}

func collectLogicalTerms(t *testing.T, n node.Node, op string) []node.Node {
	t.Helper()
	binary := requireBinaryOp(t, n, op)
	var result []node.Node
	for _, candidate := range []node.Node{binary.X, binary.Y} {
		if nested, ok := candidate.(*expr.Binary); ok && strings.EqualFold(strings.TrimSpace(nested.Op), op) {
			result = append(result, collectLogicalTerms(t, nested, op)...)
			continue
		}
		result = append(result, candidate)
	}
	return result
}

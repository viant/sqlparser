package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestParenthesizedPredicateValues(t *testing.T) {
	placeholder := expr.Value{Placeholder: true}
	integer := expr.Value{Value: 123, Kind: "int"}
	decimal := expr.Value{Value: 123.5, Kind: "numeric"}
	text := expr.Value{Value: "abc", Kind: "string"}
	for _, tc := range []struct {
		rhs    string
		values []expr.Value
	}{
		{"(?)", []expr.Value{placeholder}},
		{"(?, ?)", []expr.Value{placeholder, placeholder}},
		{"(123)", []expr.Value{integer}},
		{"(123.5)", []expr.Value{decimal}},
		{"('abc')", []expr.Value{text}},
		{"(((?)))", []expr.Value{placeholder}},
		{"(((123)))", []expr.Value{integer}},
		{"((?, ?))", []expr.Value{placeholder, placeholder}},
		{"(?, 'abc', 123.5, ?)", []expr.Value{placeholder, text, decimal, placeholder}},
		{"('abc' COLLATE nocase)", []expr.Value{text}},
		{"((?) COLLATE nocase)", []expr.Value{placeholder}},
	} {
		t.Run(tc.rhs, func(t *testing.T) {
			q, err := ParseQuery("SELECT * FROM orders WHERE order_id IN " + tc.rhs)
			require.NoError(t, err)
			binary := q.Qualify.X.(*expr.Binary)
			require.Equal(t, "IN", binary.Op)
			direct, err := expr.NewValues(binary.Y)
			require.NoError(t, err)
			require.Equal(t, tc.values, direct.X)
			ident, values, err := binary.Predicate()
			require.NoError(t, err)
			require.Equal(t, &expr.Ident{Name: "order_id"}, ident)
			require.Equal(t, tc.values, values.X)

			// Distinct bindings verify placeholder order, including mixed lists.
			var want []interface{}
			bindings := 0
			for _, value := range tc.values {
				if value.Placeholder {
					want = append(want, 100+bindings)
					bindings++
				} else {
					want = append(want, value.Value)
				}
			}
			require.Equal(t, want, values.Values(func(index int) interface{} { return 100 + index }))
			require.Equal(t, bindings, values.Idx)
		})
	}
}

func TestParenthesizedPredicateValuesRejectExpressions(t *testing.T) {
	for _, rhs := range []string{
		"(? + 1)", "(1 + ?)", "(((? + 1)))", "((? + 1) COLLATE nocase)",
		"(?, 1 + ?)", "(other_column)", "(ABS(?))", "(a[0])",
		"(SELECT order_id FROM orders)", "((SELECT order_id FROM orders))",
	} {
		t.Run(rhs, func(t *testing.T) {
			q, err := ParseQuery("SELECT * FROM orders WHERE order_id IN " + rhs)
			require.NoError(t, err)
			binary := q.Qualify.X.(*expr.Binary)
			values, err := expr.NewValues(binary.Y)
			require.ErrorContains(t, err, "unsupported value node")
			require.Nil(t, values)
			ident, values, err := binary.Predicate()
			require.Error(t, err)
			require.Nil(t, ident)
			require.Nil(t, values)
		})
	}
}

func TestParenthesizedValuesEmptyBehavior(t *testing.T) {
	for _, items := range [][]node.Node{nil, {}} {
		values, err := expr.NewValues(&expr.Parenthesis{X: items})
		require.NoError(t, err)
		require.Empty(t, values.X)
	}
	for _, raw := range []string{"()", "(?)", "(123)"} {
		// Raw SQL is not a substitute for a parsed child, even for an empty set.
		values, err := expr.NewValues(&expr.Parenthesis{Raw: raw})
		require.Error(t, err)
		require.Nil(t, values)
	}
	q, err := ParseQuery("SELECT * FROM orders WHERE order_id IN ()")
	require.NoError(t, err)
	_, _, err = q.Qualify.X.(*expr.Binary).Predicate()
	require.Error(t, err)
}

func TestNewValuesRetainsBinaryPredicateHandling(t *testing.T) {
	values, err := expr.NewValues(&expr.Binary{
		X: &expr.Ident{Name: "order_id"}, Op: "=", Y: &expr.Placeholder{Name: "?"},
	})
	require.NoError(t, err)
	require.Equal(t, []expr.Value{{Placeholder: true}}, values.X)
}

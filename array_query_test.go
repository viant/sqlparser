package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestArrayQueryArguments(t *testing.T) {
	for _, body := range []string{
		"SELECT order_id FROM orders",
		"SELECT AS STRUCT 1 AS id, 'ok' AS status",
		"SELECT AS STRUCT order_id AS id, status FROM orders WHERE order_id IN (?) ORDER BY order_id LIMIT 20",
		"select distinct as struct order_id as id from orders",
		"SELECT ALL AS STRUCT order_id FROM orders",
		" /* query */ SELECT /* mode */ AS /* type */ STRUCT order_id FROM orders",
		"SELECT/* query */AS/* mode */STRUCT/* projection */order_id FROM orders",
		"SELECT DISTINCT/* distinct */AS STRUCT order_id FROM orders",
		"SELECT ALL/* all */AS STRUCT order_id FROM orders",
		"WITH/* cte */selected AS/* body */(/* query */SELECT order_id FROM orders)/* main */SELECT AS STRUCT order_id FROM selected",
		"WITH selected AS (SELECT order_id FROM orders) SELECT AS STRUCT order_id FROM selected",
		"SELECT AS STRUCT order_id FROM orders UNION ALL SELECT AS STRUCT order_id FROM archived_orders",
		"SELECT AS STRUCT order_id FROM orders WHERE order_id = outer_orders.order_id",
	} {
		for _, enclosed := range []string{body, "(" + body + ")", "((" + body + "))"} {
			t.Run(enclosed, func(t *testing.T) {
				SQL := "SELECT TO_JSON_STRING(ARRAY(" + enclosed + ")) AS diagnostics FROM orders outer_orders"
				parsed, err := ParseQuery(SQL)
				require.NoError(t, err)
				require.Len(t, parsed.List, 1)
				require.Equal(t, "diagnostics", parsed.List[0].Alias)
				jsonCall := parsed.List[0].Expr.(*expr.Call)
				require.Len(t, jsonCall.Args, 1)
				arrayCall := jsonCall.Args[0].(*expr.Call)
				require.Len(t, arrayCall.Args, 1)
				inner := arrayCall.Args[0].(*query.Select)
				require.NotEmpty(t, inner.List)
				found := false
				Traverse(parsed, func(n node.Node) bool {
					if n == inner {
						found = true
					}
					return true
				})
				require.True(t, found, "traversal must reach the ARRAY query")

				// Render the query AST directly, independent of the call's Raw.
				innerSQL := (Stringifier{PreserveWindow: true}).String(inner)
				again, err := ParseQuery(innerSQL)
				require.NoError(t, err)
				require.Equal(t, inner.Kind, again.Kind)
				require.Equal(t, len(inner.List), len(again.List))
				require.Equal(t, innerSQL, (Stringifier{PreserveWindow: true}).String(again))
				rendered := Stringify(parsed)
				reparsed, err := ParseQuery(rendered)
				require.NoError(t, err)
				require.Equal(t, rendered, Stringify(reparsed))
			})
		}
	}
}

func TestArrayQueryPreservesClauses(t *testing.T) {
	call, err := ParseCallExpr("ARRAY(SELECT AS STRUCT order_id AS id, status FROM orders WHERE order_id IN (?) ORDER BY order_id LIMIT 20)")
	require.NoError(t, err)
	inner := call.Args[0].(*query.Select)
	require.Equal(t, "AS STRUCT", inner.Kind)
	require.Len(t, inner.List, 2)
	require.Equal(t, "id", inner.List[0].Alias)
	require.Equal(t, "orders", Stringify(inner.From.X))
	require.NotNil(t, inner.Qualify)
	require.Len(t, inner.OrderBy, 1)
	require.Equal(t, "20", inner.Limit.Value)
	ident, values, err := inner.Qualify.X.(*expr.Binary).Predicate()
	require.NoError(t, err)
	require.Equal(t, "order_id", Stringify(ident))
	require.Equal(t, []expr.Value{{Placeholder: true}}, values.X)
}

func TestArrayQueryRejectsIncompleteArguments(t *testing.T) {
	for _, body := range []string{
		"SELECT", "SELECT FROM orders", "SELECT AS", "SELECT AS STRUCT", "SELECT AS STRUCT FROM orders",
		"SELECT AS STRUCT order_id + FROM orders", "SELECT AS STRUCTURE order_id FROM orders",
		"WITH selected AS (SELECT AS STRUCT FROM orders) SELECT AS STRUCT order_id FROM selected",
		"SELECT AS STRUCT order_id FROM orders UNION ALL SELECT AS STRUCT FROM orders",
		"(SELECT AS STRUCT order_id FROM orders), 2", "(SELECT AS STRUCT order_id FROM orders) + 1",
	} {
		t.Run(body, func(t *testing.T) {
			_, err := ParseCallExpr("ARRAY(" + body + ")")
			require.Error(t, err)
			_, err = ParseQuery("SELECT TO_JSON_STRING(ARRAY(" + body + ")) AS diagnostics")
			require.Error(t, err)
		})
	}
	for _, SQL := range []string{"SELECT AS STRUCT", "SELECT AS STRUCT FROM orders", "SELECT AS STRUCTURE order_id FROM orders"} {
		_, err := ParseQuery(SQL)
		require.Error(t, err, SQL)
	}
}

func TestArrayScalarArgumentsRemainSupported(t *testing.T) {
	for _, expression := range []string{"ARRAY(1, 2)", "ARRAY(selected_id)", "ARRAY(with_value)", "ARRAY((selected_id))"} {
		parsed, err := ParseQuery("SELECT " + expression)
		require.NoError(t, err)
		require.Len(t, parsed.List, 1)
		require.Equal(t, expression, Stringify(parsed.List[0].Expr))
	}
}

func TestArrayQueryStripCollate(t *testing.T) {
	SQL := "SELECT TO_JSON_STRING(ARRAY(SELECT AS STRUCT status COLLATE nocase AS status FROM orders)) AS diagnostics"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.False(t, strings.Contains(strings.ToUpper(stripped), "COLLATE"))
	require.Contains(t, stripped, "SELECT AS STRUCT status AS status FROM orders")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

func TestArrayQueryStripCollatePreservesPagination(t *testing.T) {
	for _, body := range []string{
		"SELECT AS STRUCT status COLLATE nocase FROM orders ORDER BY order_id LIMIT 20 OFFSET 2",
		"(SELECT AS STRUCT status COLLATE nocase FROM orders ORDER BY order_id LIMIT 20 OFFSET 2)",
		"WITH selected AS (SELECT status COLLATE nocase FROM orders LIMIT 7 OFFSET 1) SELECT AS STRUCT status FROM selected LIMIT 20 OFFSET 2",
		"SELECT AS STRUCT status FROM (SELECT status COLLATE nocase FROM orders LIMIT 7 OFFSET 1) selected LIMIT 20 OFFSET 2",
		"SELECT AS STRUCT (SELECT status COLLATE nocase FROM orders LIMIT 7 OFFSET 1) AS status FROM orders LIMIT 20 OFFSET 2",
		"SELECT AS STRUCT status COLLATE nocase FROM orders LIMIT 20 OFFSET 2 UNION ALL SELECT AS STRUCT status FROM archive LIMIT 7 OFFSET 1",
	} {
		t.Run(body, func(t *testing.T) {
			SQL := "SELECT TO_JSON_STRING(ARRAY(" + body + ")) AS diagnostics"
			stripped, err := StripCollate(SQL)
			require.NoError(t, err)
			require.NotContains(t, stripped, "COLLATE")
			require.Contains(t, stripped, "LIMIT 20 OFFSET 2")
			if strings.Contains(body, "LIMIT 7") {
				require.Contains(t, stripped, "LIMIT 7 OFFSET 1")
			}
			again, err := ParseQuery(stripped)
			require.NoError(t, err)
			inner := again.List[0].Expr.(*expr.Call).Args[0].(*expr.Call).Args[0].(*query.Select)
			require.Equal(t, "20", inner.Limit.Value)
			require.Equal(t, "2", inner.Offset.Value)
		})
	}
}

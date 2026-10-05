package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestQualifyJoinedWindowSubquery(t *testing.T) {
	const SQL = `SELECT e.category, e.total
FROM events e
LEFT JOIN (
    SELECT id
    FROM labels
    QUALIFY ROW_NUMBER() OVER (
        PARTITION BY id
        ORDER BY updated_at DESC
    ) = 1
) l ON l.id = e.id`
	for _, strict := range []bool{false, true} {
		var opts []Option
		if strict {
			opts = append(opts, WithStructuralValidation())
		}
		parsed, err := ParseQuery(SQL, opts...)
		require.NoError(t, err)
		for i := 0; i < 2; i++ {
			require.Len(t, parsed.List, 2)
			require.Equal(t, "events", Stringify(parsed.From.X))
			require.Len(t, parsed.Joins, 1)
			join := parsed.Joins[0]
			require.Equal(t, "l", join.Alias)
			require.Equal(t, "l.id = e.id", Stringify(join.On))
			inner := join.With.(*expr.Parenthesis).X.(*query.Select)
			require.Nil(t, inner.Qualify)
			require.NotNil(t, inner.QualifyClause)
			predicate := inner.QualifyClause.X.(*expr.Binary)
			require.Equal(t, "=", predicate.Op)
			require.Equal(t, "1", Stringify(predicate.Y))
			window := predicate.X.(*expr.Window)
			require.Equal(t, "ROW_NUMBER()", Stringify(window.X))
			require.Len(t, window.PartitionBy, 1)
			require.Equal(t, "id", Stringify(window.PartitionBy[0]))
			require.Len(t, window.OrderBy, 1)
			require.Equal(t, "updated_at", Stringify(window.OrderBy[0].X))
			require.Equal(t, "DESC", window.OrderBy[0].Direction)
			visited := false
			Traverse(parsed, func(n node.Node) bool {
				if n == window {
					visited = true
				}
				return true
			})
			require.True(t, visited)
			parsed, err = ParseQuery(Stringify(parsed), opts...)
			require.NoError(t, err)
		}
	}
}

func TestQualifyWindowRoundTrip(t *testing.T) {
	for _, SQL := range []string{
		"SELECT id, ROW_NUMBER() OVER (PARTITION BY category ORDER BY updated_at DESC) AS rn FROM events QUALIFY rn = 1",
		"SELECT category, SUM(total), ROW_NUMBER() OVER (ORDER BY SUM(total) DESC) rn FROM events WHERE total > 0 GROUP BY category HAVING SUM(total) > 10 QUALIFY rn = 1 ORDER BY category LIMIT 5",
		"SELECT ROW_NUMBER() OVER () rn QUALIFY rn = 1",
		"SELECT id FROM events QUALIFY ROW_NUMBER() OVER (ORDER BY updated_at) = 1 UNION ALL SELECT id FROM archived QUALIFY ROW_NUMBER() OVER (ORDER BY updated_at DESC) = 1",
		"WITH ranked AS (SELECT id, ROW_NUMBER() OVER (ORDER BY updated_at) rn FROM events QUALIFY rn = 1) SELECT id FROM ranked",
		"SELECT id FROM (SELECT id FROM events QUALIFY ROW_NUMBER() OVER () = 1) ranked",
		"SELECT SUM(total) OVER (PARTITION BY category, region ORDER BY updated_at + 1 DESC, id ASC ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) AS running FROM events QUALIFY running > 0",
		"SELECT SUM(total) OVER (ORDER BY id RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) running FROM events QUALIFY running > 0",
		"SELECT SUM(total) OVER (ROWS 2 PRECEDING) running FROM events QUALIFY running > 0",
		"SELECT SUM(total) OVER (ROWS BETWEEN CURRENT ROW AND 2 FOLLOWING) running FROM events QUALIFY running > 0",
		"SELECT ROW_NUMBER() over /* window */ (partition /* key */ by id order by updated_at desc) rn FROM events qualify rn = 1",
		"SELECT qualify_count, `QUALIFY`, ROW_NUMBER() OVER () AS over_count FROM events AS qualify_source QUALIFY qualify_count > 0 AND over_count = 1",
		"SELECT id FROM events QUALIFY[rn] = 1",
		"SELECT id FROM events QUALIFY ROW_NUMBER() OVER () = 1 ORDER BY id + 1 DESC LIMIT 2",
		"SELECT FIRST_VALUE(name) OVER () COLLATE nocase AS name FROM events",
	} {
		t.Run(SQL, func(t *testing.T) {
			q, err := ParseQuery(SQL, WithStructuralValidation())
			require.NoError(t, err)
			renderer := Stringifier{PreserveWindow: true}
			rendered := renderer.String(q)
			again, err := ParseQuery(rendered, WithStructuralValidation())
			require.NoError(t, err)
			require.Equal(t, rendered, renderer.String(again))
			require.Equal(t, q, again, "all parsed clauses and expressions must survive the round trip")
		})
	}
}

func TestQualifyStageAndLineage(t *testing.T) {
	SQL := "SELECT category, ROW_NUMBER() OVER (ORDER BY SUM(total) DESC) rn FROM events WHERE active = 1 GROUP BY category HAVING SUM(total) > 10 QUALIFY rn = 1 ORDER BY category LIMIT 5"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	require.Equal(t, "active = 1", Stringify(q.Qualify))
	require.Equal(t, "SUM(total) > 10", Stringify(q.Having))
	require.Equal(t, "rn = 1", Stringify(q.QualifyClause))
	require.Equal(t, "rn", q.List[1].Alias)
	require.IsType(t, &expr.Window{}, q.List[1].Expr)
	rendered := (Stringifier{PreserveWindow: true}).String(q)
	previous := -1
	for _, clause := range []string{" WHERE ", " GROUP BY ", " HAVING ", " QUALIFY ", " ORDER BY category", " LIMIT "} {
		pos := strings.Index(rendered, clause)
		require.Greater(t, pos, previous, clause)
		previous = pos
	}
	lineage := (Lineage{Query: q}).Compile()
	for _, name := range []string{"active", "total", "rn"} {
		_, direct := lineage.Lookup(name)
		require.False(t, direct, name+" must not become a direct projected column")
	}
	origin, ok := lineage.Lookup("category")
	require.True(t, ok)
	require.Equal(t, ColumnOrigin{Table: "events", Column: "category"}, origin)
	union, err := ParseQuery("SELECT id FROM events QUALIFY ROW_NUMBER() OVER () = 1 UNION ALL SELECT id FROM archived QUALIFY ROW_NUMBER() OVER () = 2")
	require.NoError(t, err)
	require.Equal(t, "ROW_NUMBER() OVER () = 1", Stringify(union.QualifyClause))
	require.Equal(t, "ROW_NUMBER() OVER () = 2", Stringify(union.Union.X.QualifyClause))
}

func TestQualifyComputedOrderDirection(t *testing.T) {
	for _, direction := range []string{"ASC", "DESC"} {
		SQL := "SELECT id FROM events QUALIFY ROW_NUMBER() OVER () = 1 ORDER BY id + 1 " + direction
		q, err := ParseQuery(SQL)
		require.NoError(t, err)
		require.Len(t, q.OrderBy, 1)
		require.Equal(t, direction, q.OrderBy[0].Direction)
		require.Empty(t, q.OrderBy[0].Alias)
		require.Equal(t, SQL, Stringify(q))
	}
}

func TestQualifyNestedTraversalOrder(t *testing.T) {
	SQL := "WITH ranked AS (SELECT SUM(?) OVER (PARTITION BY COALESCE(category, ?) ORDER BY COALESCE(updated_at, ?) ROWS BETWEEN ? PRECEDING AND ? FOLLOWING) rn FROM events QUALIFY rn > ?) " +
		"SELECT ? FROM (SELECT id FROM ranked QUALIFY ROW_NUMBER() OVER (ORDER BY id + ?) > ?) d " +
		"JOIN (SELECT id FROM labels QUALIFY ROW_NUMBER() OVER () > ?) l ON l.id = ? WHERE d.id > ? " +
		"QUALIFY ROW_NUMBER() OVER (ORDER BY d.id + ?) > ? UNION ALL SELECT ?"
	q, err := ParseQuery(SQL)
	require.NoError(t, err)
	var placeholders []*expr.Placeholder
	var columns []string
	Traverse(q, func(n node.Node) bool {
		if p, ok := n.(*expr.Placeholder); ok {
			placeholders = append(placeholders, p)
		}
		if ident, ok := n.(*expr.Ident); ok {
			columns = append(columns, ident.Name)
		}
		return true
	})
	require.Len(t, placeholders, 15)
	// Compare the collected pointers to the AST in textual order, independently
	// of serialization's preservation of raw function and subquery text.
	cte := q.WithSelects[0].X
	w := cte.List[0].Expr.(*expr.Window)
	expected := []node.Node{
		w.X.(*expr.Call).Args[0], w.PartitionBy[0].(*expr.Call).Args[1], w.OrderBy[0].X.(*expr.Call).Args[1],
		w.Frame.Start.X, w.Frame.End.X, cte.QualifyClause.X.(*expr.Binary).Y, q.List[0].Expr,
	}
	derived := q.From.X.(*expr.Raw).X.(*query.Select).QualifyClause.X.(*expr.Binary)
	expected = append(expected, derived.X.(*expr.Window).OrderBy[0].X.(*expr.Binary).Y, derived.Y)
	joined := q.Joins[0].With.(*expr.Parenthesis).X.(*query.Select)
	expected = append(expected, joined.QualifyClause.X.(*expr.Binary).Y, q.Joins[0].On.X.(*expr.Binary).Y, q.Qualify.X.(*expr.Binary).Y)
	outer := q.QualifyClause.X.(*expr.Binary)
	expected = append(expected, outer.X.(*expr.Window).OrderBy[0].X.(*expr.Binary).Y, outer.Y, q.Union.X.List[0].Expr)
	for i, n := range expected {
		require.Same(t, n, placeholders[i])
	}
	require.Contains(t, columns, "category")
	require.Contains(t, columns, "updated_at")
}

func TestQualifyWindowRejectsMalformed(t *testing.T) {
	for _, SQL := range []string{
		"SELECT id FROM events WHERE QUALIFY rn = 1", "SELECT id FROM events HAVING QUALIFY rn = 1",
		"SELECT id FROM events GROUP BY QUALIFY rn = 1", "SELECT QUALIFY rn = 1",
		"SELECT id FROM events JOIN labels ON QUALIFY rn = 1",
		"SELECT id FROM events QUALIFY rn = 1 ORDER BY", "SELECT id FROM events QUALIFY rn = 1 LIMIT",
		"SELECT id FROM events GROUP BY id, QUALIFY rn = 1", "SELECT id FROM events QUALIFY rn = 1 ORDER BY id,",
		"SELECT id FROM events GROUP BY id + 1, QUALIFY rn = 1", "SELECT id FROM events QUALIFY rn = 1 ORDER BY id + 1,",
		"SELECT id FROM events QUALIFY", "SELECT id FROM events QUALIFY rn =", "SELECT id FROM events QUALIFY ORDER BY id",
		"SELECT id FROM events QUALIFY rn = 1 AND", "SELECT id FROM events QUALIFY rn = 1 QUALIFY rn = 2",
		"SELECT id FROM events ORDER BY id QUALIFY rn = 1", "SELECT id FROM events LIMIT 1 QUALIFY rn = 1",
		"SELECT id FROM events QUALIFY rn = 1 WHERE id = 2", "SELECT id FROM events QUALIFY rn = 1 HAVING id = 2",
		"SELECT id FROM events QUALIFY rn = 1 GROUP BY id", "SELECT id QUALIFY rn = 1 FROM events",
		"SELECT ROW_NUMBER() OVER", "SELECT ROW_NUMBER() OVER w FROM events", "SELECT ROW_NUMBER() OVER (w) FROM events",
		"SELECT ROW_NUMBER() OVER (PARTITION BY) FROM events", "SELECT ROW_NUMBER() OVER (ORDER BY) FROM events",
		"SELECT ROW_NUMBER() OVER (PARTITION BY id,) FROM events", "SELECT ROW_NUMBER() OVER (ORDER BY id,) FROM events",
		"SELECT ROW_NUMBER() OVER (ORDER BY id PARTITION BY id) FROM events", "SELECT ROW_NUMBER() OVER (ORDER BY id +) FROM events",
		"SELECT SUM(id) OVER (ORDER BY id ROWS) FROM events", "SELECT SUM(id) OVER (ROWS BETWEEN 1 PRECEDING) FROM events",
		"SELECT SUM(id) OVER (ROWS -1 PRECEDING) FROM events", "SELECT SUM(id) OVER (ROWS '1' PRECEDING) FROM events",
		"SELECT SUM(id) OVER (ROWS UNBOUNDED FOLLOWING) FROM events", "SELECT SUM(id) OVER (ROWS BETWEEN CURRENT ROW AND 1 PRECEDING) FROM events",
		"SELECT SUM(id) OVER (ROWS 1 FOLLOWING) FROM events", "SELECT SUM(id) OVER (ROWS BETWEEN 1 PRECEDING AND UNBOUNDED PRECEDING) FROM events",
		"SELECT ROW_NUMBER() OVER () OVER () FROM events", "SELECT id OVER () FROM events",
		"SELECT ROW_NUMBER() OVER () FROM events WINDOW w AS (ORDER BY id)",
	} {
		t.Run(SQL, func(t *testing.T) {
			_, err := ParseQuery(SQL)
			require.Error(t, err)
			_, err = ParseQuery(SQL, WithStructuralValidation())
			require.Error(t, err)
		})
	}
}

func TestQualifyWindowStripCollate(t *testing.T) {
	SQL := "SELECT category FROM events QUALIFY FIRST_VALUE(name COLLATE nocase) OVER (PARTITION BY category COLLATE nocase ORDER BY name COLLATE nocase) = 'x'"
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	q, err := ParseQuery(stripped)
	require.NoError(t, err)
	require.NotNil(t, q.QualifyClause)
	require.IsType(t, &expr.Window{}, q.QualifyClause.X.(*expr.Binary).X)
}

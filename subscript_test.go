package sqlparser

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"testing"
)

func TestSubscript(t *testing.T) {
	for _, tc := range []struct {
		sql   string
		count int
	}{
		{"IAB[0]", 1},
		{"v.IAB[SAFE_OFFSET(0)]", 1},
		{"`IAB`[OFFSET(1)]", 1},
		{"IAB[SAFE_ORDINAL(1)]", 1},
		{"IAB[positions[0]]", 2},
		{"matrix[0][1]", 2},
		{"make_array(x)[OFFSET(0)]", 1},
		{"(IAB)[OFFSET(0)]", 1},
		{"IAB[IF(x = ']', 0, 1)]", 1},
		{"IAB[0] + 1", 1},
		{"IFNULL(STRING_AGG(DISTINCT IAB[SAFE_OFFSET(0)], ', ' LIMIT 20), '')", 1},
		{"STRING_AGG(DISTINCT IAB[0], ', ' ORDER BY IAB[0] LIMIT 20)", 2},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			parsed, err := ParseQuery("SELECT " + tc.sql + " AS result FROM src v")
			require.NoError(t, err)
			assertCount := func(n node.Node) {
				count := 0
				Traverse(n, func(n node.Node) bool {
					if value, ok := n.(*expr.Subscript); ok {
						count++
						require.NotNil(t, value.X)
						require.NotNil(t, value.Index)
					}
					return true
				})
				require.Equal(t, tc.count, count)
			}
			assertCount(parsed)
			rendered := Stringify(parsed)
			again, err := ParseQuery(rendered)
			require.NoError(t, err, rendered)
			assertCount(again)
			require.Equal(t, rendered, Stringify(again))
		})
	}
}

func TestSubscriptTreeAndLineage(t *testing.T) {
	q, err := ParseQuery("SELECT v.IAB[SAFE_OFFSET(position)] AS value, v.id FROM src v")
	require.NoError(t, err)
	sub, ok := q.List[0].Expr.(*expr.Subscript)
	require.True(t, ok)
	require.Equal(t, "v.IAB", Stringify(sub.X))
	call, ok := sub.Index.(*expr.Call)
	require.True(t, ok)
	require.Equal(t, "SAFE_OFFSET", Stringify(call.X))
	require.Len(t, call.Args, 1)
	require.Equal(t, "position", Stringify(call.Args[0]))
	var names []string
	Traverse(sub, func(n node.Node) bool {
		if id, ok := n.(*expr.Ident); ok {
			names = append(names, id.Name)
		}
		return true
	})
	require.Contains(t, names, "position")
	require.NotEmpty(t, NewColumn(q.List[0]).Expression)
	_, direct := (Lineage{Query: q}).Compile().Lookup("value")
	require.False(t, direct, "array access must not inherit scalar source-column provenance")
	_, direct = (Lineage{Query: q}).Compile().Lookup("id")
	require.True(t, direct)
}

func TestSubscriptRejectsMalformedIndex(t *testing.T) {
	for _, expression := range []string{"a[]", "a[ ]", "a[0", "a[0,1]", "a[0 junk]", "a[0+]", "a[b[]]", "a[SAFE_OFFSET(0) junk]"} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression + " AS result FROM src")
			require.Error(t, err)
		})
	}
}

func TestSubscriptStripCollate(t *testing.T) {
	actual, err := StripCollate("SELECT a[(idx COLLATE nocase)] AS result FROM src")
	require.NoError(t, err)
	require.NotContains(t, actual, "COLLATE")
	_, err = ParseQuery(actual)
	require.NoError(t, err)
}

func TestSubscriptCollateRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		expression string
		canonical  string
		stripped   string
	}{
		{"a[idx COLLATE nocase ]", "a[idx COLLATE nocase]", "a[idx]"},
		{"a[idx COLLATE nocase]", "a[idx COLLATE nocase]", "a[idx]"},
		{"a[b[0] COLLATE nocase]", "a[b[0] COLLATE nocase]", "a[b[0]]"},
		{"a[b[idx COLLATE nocase]]", "a[b[idx COLLATE nocase]]", "a[b[idx]]"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			SQL := "SELECT " + tc.expression + " AS value FROM src"
			for i := 0; i < 2; i++ {
				q, err := ParseQuery(SQL, WithStructuralValidation())
				require.NoError(t, err)
				collations := 0
				Traverse(q.List[0].Expr, func(n node.Node) bool {
					if c, ok := n.(*expr.Collate); ok {
						collations++
						require.Equal(t, "nocase", c.Collation)
					}
					return true
				})
				require.Equal(t, 1, collations)
				SQL = Stringify(q)
				require.Equal(t, "SELECT "+tc.canonical+" AS value FROM src", SQL)
			}
			stripped, err := StripCollate(SQL)
			require.NoError(t, err)
			require.Equal(t, "SELECT "+tc.stripped+" AS value FROM src", stripped)
		})
	}
}

func TestSubscriptAggregateTree(t *testing.T) {
	text := "IFNULL(STRING_AGG(DISTINCT IAB[SAFE_OFFSET(0)], ', ' LIMIT 20), '')"
	q, err := ParseQuery("SELECT " + text + " AS result FROM src")
	require.NoError(t, err)
	outer := q.List[0].Expr.(*expr.Call)
	agg := outer.Args[0].(*expr.Call)
	require.Equal(t, "STRING_AGG", Stringify(agg.X))
	distinct := agg.Args[0].(*expr.Unary)
	require.Equal(t, "DISTINCT", distinct.Op)
	require.IsType(t, &expr.Subscript{}, distinct.X)
	limit := agg.Args[1].(*expr.Binary)
	require.Equal(t, "LIMIT", limit.Op)
	require.Equal(t, "20", Stringify(limit.Y))
	require.Equal(t, text, Stringify(outer))
}

func TestSubscriptPreservesBracketAlias(t *testing.T) {
	q, err := ParseQuery("SELECT id [alias], COUNT(*) [count] FROM records GROUP BY id")
	require.NoError(t, err)
	require.Equal(t, "[alias]", q.List[0].Alias)
	require.Equal(t, "[count]", q.List[1].Alias)
	require.IsType(t, &expr.Ident{}, q.List[0].Expr)
}

func TestSubscriptRejectsFieldSuffix(t *testing.T) {
	for _, expression := range []string{
		"a[SAFE_OFFSET(0)].field", "a[0] .field", "a[0] /* comment */ .field",
		"a[0][1].field", "a[b[0].field]", "COALESCE(a[0].field, 0)",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseQuery("SELECT " + expression + " AS value FROM src WHERE id = 1")
			require.ErrorContains(t, err, "unsupported field access after subscript")
		})
	}
}

func TestSubscriptStructuralValidation(t *testing.T) {
	for _, expression := range []string{
		"a[b[0]]", "a[0][1]", "a[IF(x = ']', 0, 1)]",
		"`a`[OFFSET(0)]", "a[0 /* ] */]", "a[0 -- ]\n]",
	} {
		t.Run(expression, func(t *testing.T) {
			SQL := "SELECT " + expression + " AS value FROM src WHERE id = 1"
			q, err := ParseQuery(SQL, WithStructuralValidation())
			require.NoError(t, err)
			require.IsType(t, &expr.Subscript{}, q.List[0].Expr)
			require.Equal(t, "value", q.List[0].Alias)
			require.Equal(t, "src", Stringify(q.From.X))
			require.NotNil(t, q.Qualify)
			_, err = ParseQuery(Stringify(q), WithStructuralValidation())
			require.NoError(t, err)
			call, err := ParseCallExpr("COALESCE(" + expression + ", 0)")
			require.NoError(t, err)
			require.Len(t, call.Args, 2)
			require.IsType(t, &expr.Subscript{}, call.Args[0])
		})
	}
	q, err := ParseQuery("SELECT id [a]]b], COUNT(*) [count] FROM records", WithStructuralValidation())
	require.NoError(t, err)
	require.Equal(t, "[a]]b]", q.List[0].Alias)
	require.Equal(t, "[count]", q.List[1].Alias)
}

func TestSubscriptProjectionSuffixBoundary(t *testing.T) {
	for _, expression := range []string{
		"(a[0]).field", "((a[0])).field", "(a[0]) /* comment */ .field",
		"a[0]::text", "(a[0])::text", "COALESCE(a[0], 0)::text",
		"a[0]]", "a[0] + 1::text",
	} {
		for _, format := range []string{
			"SELECT %s AS value FROM src WHERE id = 1",
			"WITH c AS (SELECT %s FROM src) SELECT * FROM c",
			"SELECT * FROM (SELECT %s FROM src) c",
			"SELECT id FROM src UNION ALL SELECT %s FROM src",
		} {
			SQL := fmt.Sprintf(format, expression)
			t.Run(SQL, func(t *testing.T) {
				_, err := ParseQuery(SQL)
				require.Error(t, err)
			})
		}
	}
}

func TestAdjacentKeywordBracketQuotes(t *testing.T) {
	for _, alias := range []string{"[a--b]", "[a]]b]", "[$INDEX]"} {
		for _, keyword := range []string{"AS", "as"} {
			SQL := "SELECT id " + keyword + alias + " FROM[src--table] WHERE id = 1"
			t.Run(SQL, func(t *testing.T) {
				for _, opts := range [][]Option{nil, {WithStructuralValidation()}} {
					q, err := ParseQuery(SQL, opts...)
					require.NoError(t, err)
					require.Equal(t, alias, q.List[0].Alias)
					require.NotNil(t, q.From.X)
					require.NotNil(t, q.Qualify)
					require.Contains(t, Stringify(q), "[src--table]")
				}
			})
		}
	}
}

func TestSubscriptParameterRoundTrip(t *testing.T) {
	for _, parameter := range []string{"?", ":index", "$index", "${index}"} {
		for _, expression := range []string{
			"a[" + parameter + "]", "a[" + parameter + " ]", "a[b[" + parameter + "]]",
			"a[" + parameter + "][0]", "a[OFFSET(" + parameter + ")]",
		} {
			t.Run(expression, func(t *testing.T) {
				SQL := "SELECT " + expression + " AS value FROM src"
				for i := 0; i < 2; i++ {
					q, err := ParseQuery(SQL, WithStructuralValidation())
					require.NoError(t, err)
					var parameters []string
					Traverse(q, func(n node.Node) bool {
						if p, ok := n.(*expr.Placeholder); ok {
							parameters = append(parameters, p.Name)
						}
						return true
					})
					require.Equal(t, []string{parameter}, parameters)
					require.Equal(t, "value", q.List[0].Alias)
					require.NotNil(t, q.From.X)
					SQL = Stringify(q)
				}
			})
		}
	}
}

func TestContextualKeywordSubscripts(t *testing.T) {
	for _, name := range []string{"values", "by", "update", "table", "returning", "order", "group"} {
		for _, expression := range []string{name + "[b[0]]", "a[" + name + "[0]]", name + "[$index]"} {
			SQL := "SELECT " + expression + " AS value FROM src"
			t.Run(SQL, func(t *testing.T) {
				q, err := ParseQuery(SQL, WithStructuralValidation())
				require.NoError(t, err)
				require.Equal(t, SQL, Stringify(q))
			})
		}
	}
}

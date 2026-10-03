package sqlparser

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
)

func TestTraverseJoins(t *testing.T) {
	for _, tc := range []struct {
		name  string
		SQL   string
		joins int
		on    int
	}{
		{"cross", "SELECT a.id FROM a CROSS JOIN b", 1, 0},
		{"unnest", "SELECT a.id FROM a CROSS JOIN UNNEST(a.items) item", 1, 0},
		{"mixed", "SELECT a.id FROM a CROSS JOIN b JOIN c ON c.id = b.id CROSS JOIN UNNEST(a.items) item LEFT JOIN d ON d.id = c.id", 4, 2},
		{"inner", "SELECT a.id FROM a JOIN b ON b.id = a.id", 1, 1},
		{"left", "SELECT a.id FROM a LEFT JOIN b ON b.id = a.id", 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseQuery(tc.SQL)
			require.NoError(t, err)
			require.Len(t, q.Joins, tc.joins)
			visited := make(map[node.Node]bool)
			onCount := 0
			require.NotPanics(t, func() {
				Traverse(q, func(n node.Node) bool {
					require.NotNil(t, n, "visitor must never receive an absent node")
					if reflect.TypeOf(n).Kind() == reflect.Ptr {
						visited[n] = true
					}
					if _, ok := n.(*expr.Qualify); ok {
						onCount++
					}
					return true
				})
			})
			require.Equal(t, tc.on, onCount)
			for _, join := range q.Joins {
				require.True(t, visited[join], "join must be visited")
				require.True(t, visited[join.With], "joined source must be visited")
				if call, ok := join.With.(*expr.Call); ok {
					for _, arg := range call.Args {
						require.True(t, visited[arg], "UNNEST argument must be visited")
					}
				}
				if join.On != nil {
					require.True(t, visited[join.On])
					require.True(t, visited[join.On.X])
					predicate := join.On.X.(*expr.Binary)
					require.True(t, visited[predicate.X])
					require.True(t, visited[predicate.Y])
				}
			}
		})
	}
}

func TestTraverseAbsentNodes(t *testing.T) {
	for _, n := range []node.Node{nil, (*expr.Qualify)(nil), (*expr.Ident)(nil), (*query.Join)(nil), (*query.Select)(nil)} {
		t.Run(fmt.Sprintf("%T", n), func(t *testing.T) {
			called := false
			require.NotPanics(t, func() {
				Traverse(n, func(node.Node) bool {
					called = true
					return true
				})
			})
			require.False(t, called, "absent nodes must be skipped before visiting")
		})
	}
}

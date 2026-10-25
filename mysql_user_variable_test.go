package sqlparser

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"testing"
)

func TestMySQLUserVariableAssignment(t *testing.T) {
	for _, sql := range []string{
		"SELECT (@i:=@i+1) AS POS FROM records",
		"SELECT @i:=0 AS initial",
		"SELECT @a:=@b:=1+2*3 AS value",
		"SELECT (@i:=@i+1) AS POS, t.TARGET_KEY FROM (SELECT TARGET_KEY FROM records) t, (SELECT @i:=0) AS p",
	} {
		t.Run(sql, func(t *testing.T) {
			q, err := ParseQuery(sql, mysqlVariableExtension())
			require.NoError(t, err)
			again, err := ParseQuery(Stringify(q), mysqlVariableExtension())
			require.NoError(t, err)
			require.Equal(t, Stringify(q), Stringify(again))
		})
	}
	q, err := ParseQuery("SELECT @a:=@b:=1+2*3 AS value", mysqlVariableExtension())
	require.NoError(t, err)
	assignment := requireBinaryOp(t, q.List[0].Expr, ":=")
	require.Equal(t, "@a", Stringify(assignment.X))
	nested := requireBinaryOp(t, assignment.Y, ":=")
	require.Equal(t, "@b", Stringify(nested.X))
	sum := requireBinaryOp(t, nested.Y, "+")
	requireBinaryOp(t, sum.Y, "*")
	require.IsType(t, &expr.Ident{}, assignment.X)
	for _, sql := range []string{"SELECT @ AS value", "SELECT @i:= AS value"} {
		_, err := ParseQuery(sql, mysqlVariableExtension())
		require.Error(t, err)
	}
}

// Variable operands use the existing opt-in extension contract. Default syntax
// must continue to reject opaque @values operands used by extension callers.
func mysqlVariableExtension() Option {
	return WithErrorHandler(func(err error, cursor *parsly.Cursor, dest any) error {
		out, ok := dest.(*node.Node)
		if !ok || cursor.Pos >= len(cursor.Input) || cursor.Input[cursor.Pos] != '@' {
			return err
		}
		start := cursor.Pos
		end := start + 1
		for end < len(cursor.Input) && (cursor.Input[end] >= 'a' && cursor.Input[end] <= 'z' || cursor.Input[end] >= 'A' && cursor.Input[end] <= 'Z' || cursor.Input[end] >= '0' && cursor.Input[end] <= '9' || cursor.Input[end] == '_') {
			end++
		}
		if end == start+1 {
			return err
		}
		*out = &expr.Ident{Name: string(cursor.Input[start:end])}
		cursor.Pos = end
		return nil
	})
}

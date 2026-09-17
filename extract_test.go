package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
)

func TestExtractRoundTrip(t *testing.T) {
	for _, sql := range []string{
		"SELECT EXTRACT(HOUR FROM sp.hstamp) AS event_hour FROM signals sp",
		"SELECT extract(day from COALESCE(t.stamp, CURRENT_TIMESTAMP())) AS d FROM t",
		"SELECT IFNULL(EXTRACT(YEAR FROM t.stamp), 0) AS y FROM t",
	} {
		q, err := ParseQuery(sql)
		if err != nil {
			t.Fatal(err)
		}
		got := Stringify(q)
		if got != sql {
			t.Fatalf("got %q, want %q", got, sql)
		}
		if _, err := ParseQuery(got); err != nil {
			t.Fatal(err)
		}
	}
	call, err := ParseCallExpr("EXTRACT(HOUR FROM sp.hstamp)")
	if err != nil {
		t.Fatal(err)
	}
	arg, ok := call.Args[0].(*expr.Binary)
	if !ok {
		t.Fatalf("missing structured argument: %#v", call.Args)
	}
	if _, ok := arg.X.(*expr.Raw); !ok {
		t.Fatal("date part must not be a column")
	}
	if got := Stringify(arg.Y); got != "sp.hstamp" {
		t.Fatalf("operand: %s", got)
	}
	for _, sql := range []string{"EXTRACT()", "EXTRACT(HOUR)", "EXTRACT(FROM stamp)", "EXTRACT(HOUR FROM)", "EXTRACT(HOUR FROM x, y)", "EXTRACT(HOUR FROM x trailing)", "EXTRACT(HOUR FROM x +)", "EXTRACT(HOUR FROM x FROM y)"} {
		if _, err := ParseCallExpr(sql); err == nil {
			t.Fatalf("accepted %s", sql)
		}
	}
}

func TestExtractWeekdayAndComments(t *testing.T) {
	for _, part := range []string{
		"WEEK(SUNDAY)", "WEEK(MONDAY)", "WEEK(TUESDAY)", "WEEK(WEDNESDAY)",
		"WEEK(THURSDAY)", "WEEK(FRIDAY)", "WEEK(SATURDAY)", "week(monday)",
		"/*leading*/ HOUR", "HOUR /*trailing*/", "HOUR--trailing\n",
		"/*leading*/ WEEK/*week*/(/*day*/MONDAY/*end*/)/*trailing*/",
	} {
		t.Run(part, func(t *testing.T) {
			call, err := ParseCallExpr("EXTRACT(" + part + " FROM t.stamp)")
			require.NoError(t, err)
			require.Len(t, call.Args, 1)
			arg := call.Args[0].(*expr.Binary)
			require.IsType(t, &expr.Raw{}, arg.X)
			require.Equal(t, "t.stamp", Stringify(arg.Y))
			SQL := "SELECT EXTRACT(" + part + " FROM t.stamp) AS value FROM events t"
			parsed, err := ParseQuery(SQL)
			require.NoError(t, err)
			again, err := ParseQuery(Stringify(parsed))
			require.NoError(t, err)
			require.Equal(t, Stringify(parsed), Stringify(again))
			// Also render the native argument, without relying on Call.Raw.
			rebuilt, err := ParseCallExpr("EXTRACT(" + Stringify(arg) + ")")
			require.NoError(t, err)
			require.Equal(t, Stringify(arg), Stringify(rebuilt.Args[0]))
		})
	}
}

func TestExtractTimeZone(t *testing.T) {
	for _, body := range []string{
		"DATE FROM t.stamp AT TIME ZONE 'America/Los_Angeles'",
		"hour from t.stamp at time zone 'UTC'",
		"WEEK(MONDAY) FROM COALESCE(t.stamp, CURRENT_TIMESTAMP()) AT TIME ZONE COALESCE(?, 'UTC')",
		"HOUR FROM t.stamp AT/*a*/TIME/*b*/ZONE/*c*/t.zone",
		"HOUR FROM t.stamp AT--a\nTIME ZONE ('UTC')",
		"HOUR FROM t.stamp AT TIME ZONE (SELECT zone FROM config LIMIT 1)",
	} {
		t.Run(body, func(t *testing.T) {
			call, err := ParseCallExpr("EXTRACT(" + body + ")")
			require.NoError(t, err)
			arg := call.Args[0].(*expr.Binary)
			zone, ok := arg.Y.(*expr.Binary)
			require.True(t, ok, "missing time-zone AST")
			require.NotNil(t, zone.X)
			require.NotNil(t, zone.Y)
			rebuilt, err := ParseCallExpr("EXTRACT(" + Stringify(arg) + ")")
			require.NoError(t, err)
			require.Equal(t, Stringify(arg), Stringify(rebuilt.Args[0]))
			parsed, err := ParseQuery("SELECT EXTRACT(" + body + ") AS value FROM events t")
			require.NoError(t, err)
			again, err := ParseQuery(Stringify(parsed))
			require.NoError(t, err)
			require.Equal(t, Stringify(parsed), Stringify(again))
		})
	}
}

func TestExtractTraversalAndTransformation(t *testing.T) {
	SQL := "SELECT EXTRACT(/*part*/ WEEK(/*day*/MONDAY) FROM t.stamp AT TIME ZONE COALESCE(t.zone COLLATE nocase, ?)) AS value FROM events t"
	parsed, err := ParseQuery(SQL)
	require.NoError(t, err)
	var selectors []string
	placeholders := 0
	Traverse(parsed, func(n node.Node) bool {
		switch actual := n.(type) {
		case *expr.Selector:
			selectors = append(selectors, Stringify(actual))
		case *expr.Placeholder:
			placeholders++
		case *expr.Ident:
			require.NotEqual(t, "WEEK", actual.Name)
			require.NotEqual(t, "MONDAY", actual.Name)
		}
		return true
	})
	require.Contains(t, selectors, "t.stamp")
	require.Contains(t, selectors, "t.zone")
	require.Equal(t, 1, placeholders)
	stripped, err := StripCollate(SQL)
	require.NoError(t, err)
	require.NotContains(t, stripped, "COLLATE")
	require.Contains(t, stripped, "WEEK(MONDAY)")
	require.Contains(t, stripped, "AT TIME ZONE COALESCE(t.zone, ?)")
	_, err = ParseQuery(stripped)
	require.NoError(t, err)
}

func TestExtractRejectsMalformedWeekdayAndTimeZone(t *testing.T) {
	for _, body := range []string{
		"/*empty*/ FROM t.stamp", "HO/*split*/UR FROM t.stamp", "HOUR extra FROM t.stamp",
		"WEEK() FROM t.stamp", "WEEK(FUNDAY) FROM t.stamp", "WEEK(MONDAY, TUESDAY) FROM t.stamp",
		"WEEK(MONDAY + 1) FROM t.stamp", "WEEK(MONDAY) extra FROM t.stamp", "HOUR(MONDAY) FROM t.stamp",
		"HOUR FROM t.stamp AT", "HOUR FROM t.stamp AT TIME", "HOUR FROM t.stamp AT TIME ZONE",
		"HOUR FROM t.stamp AT TIME ZONE /*empty*/", "HOUR FROM t.stamp AT TIME ZONE ? +",
		"HOUR FROM t.stamp AT TIME ZONE 'UTC' trailing", "HOUR FROM t.stamp AT TIME ZONE 'UTC', 'PST'",
		"HOUR FROM t.stamp AT TIME ZONE 'UTC' AT TIME ZONE 'PST'", "HOUR FROM AT TIME ZONE 'UTC'",
	} {
		_, err := ParseCallExpr("EXTRACT(" + body + ")")
		require.Error(t, err, body)
	}
}

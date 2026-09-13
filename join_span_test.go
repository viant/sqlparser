package sqlparser

import "testing"

func TestJoinSourceSpans(t *testing.T) {
	for _, source := range []string{
		"SELECT u.*,a.* FROM users u JOIN accounts a ON u.id = a.user_id WHERE u.id=1",
		"SELECT u.*,a.* FROM users u -- root comment\nJOIN accounts a ON u.id = a.user_id",
		"SELECT u.*,a.*,p.* FROM users u JOIN accounts a ON u.id=a.user_id LEFT JOIN profiles p ON a.id=p.account_id ORDER BY u.id",
	} {
		t.Run(source, func(t *testing.T) {
			parsed, err := ParseQuery(source)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Joins) == 0 {
				t.Fatal("missing joins")
			}
			for _, join := range parsed.Joins {
				if join.Span.End <= join.Span.Begin || int(join.Span.End) > len(source) {
					t.Fatalf("invalid join span %+v", join.Span)
				}
				if join.OnSpan.Begin < join.Span.Begin || join.OnSpan.End > join.Span.End || join.OnSpan.End <= join.OnSpan.Begin {
					t.Fatalf("invalid ON span %+v within %+v", join.OnSpan, join.Span)
				}
				predicate := source[join.OnSpan.Begin:join.OnSpan.End]
				if predicate != "ON u.id = a.user_id" && predicate != "ON u.id=a.user_id" && predicate != "ON a.id=p.account_id" {
					t.Fatalf("predicate span %q", predicate)
				}
			}
		})
	}
}

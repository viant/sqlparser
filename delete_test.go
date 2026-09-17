package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDelete(t *testing.T) {
	testcases := []struct {
		SQL         string
		expect      string
		description string
	}{
		{
			description: "basic",
			SQL:         "DELETE FROM PRODUCTS",
			expect:      "DELETE FROM PRODUCTS",
		},
		{
			description: "with qualify",
			SQL:         "DELETE FROM PRODUCTS WHERE ID = 10",
			expect:      "DELETE FROM PRODUCTS WHERE ID = 10",
		},
		{
			description: "with aliases",
			SQL:         "DELETE p FROM PRODUCTS p WHERE p.ID = 10",
			expect:      "DELETE p FROM PRODUCTS p WHERE p.ID = 10",
		},
		{
			description: "with joins",
			SQL:         "DELETE p, o FROM PRODUCTS p JOIN OTHER o ON p.ID = o.PRODUCT_ID WHERE p.ID = 10",
			expect:      "DELETE p,  o FROM PRODUCTS p JOIN OTHER o ON p.ID = o.PRODUCT_ID WHERE p.ID = 10",
		},
	}

	//for _, testcase := range testcases[len(testcases)-1:] {
	for _, testcase := range testcases {
		statement, err := ParseDelete(testcase.SQL)
		if !assert.Nil(t, err, testcase.description) {
			continue
		}

		assert.Equal(t, testcase.expect, Stringify(statement), testcase.description)
	}
}

func TestParseDeletePopulatesJoinSourceSpans(t *testing.T) {
	sql := "DELETE p FROM PRODUCTS p JOIN OTHER o ON p.ID = o.PRODUCT_ID WHERE p.ID = 10"
	statement, err := ParseDelete(sql)
	if err != nil {
		t.Fatalf("ParseDelete() error = %v", err)
	}
	if len(statement.Joins) != 1 {
		t.Fatalf("joins = %d, want 1", len(statement.Joins))
	}
	join := statement.Joins[0]
	joinStart := strings.Index(sql, "JOIN OTHER")
	onStart := strings.Index(sql, "ON p.ID")
	onEnd := strings.Index(sql, " WHERE")
	if int(join.Begin) != joinStart || int(join.End) != onEnd || int(join.OnSpan.Begin) != onStart || int(join.OnSpan.End) != onEnd {
		t.Fatalf("spans = join[%d:%d] on[%d:%d], want join[%d:%d] on[%d:%d]", join.Begin, join.End, join.OnSpan.Begin, join.OnSpan.End, joinStart, onEnd, onStart, onEnd)
	}
}

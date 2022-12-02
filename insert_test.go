package sqlparser_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlx/metadata/ast/parser"
	"github.com/viant/toolbox"
	"testing"
)

func TestParseInsert(t *testing.T) {

	var testCases = []struct {
		description string
		SQL         string
		expect      string
	}{

		{
			description: "basic insert",
			SQL:         "INSERT INTO t (c1, c2) VALUES(1, 2)",
			expect:      "INSERT INTO t (c1, c2) VALUES(1, 2)",
		},
		{
			description: "basic insert",
			SQL:         "INSERT INTO CI_AD_ORDER(ID, NAME) VALUES(0, $Name);",
			expect:      "INSERT INTO CI_AD_ORDER (ID, NAME) VALUES(0, $Name)",
		},
	}

	//for _, testCase := range testCases[len(testCases)-1:] {
	for _, testCase := range testCases {
		update, err := parser.ParseInsert(testCase.SQL)
		if !assert.Nil(t, err) {
			fmt.Printf("%v\n", testCase.SQL)
			continue
		}

		actual := parser.Stringify(update)
		if !assert.EqualValues(t, testCase.expect, actual) {
			toolbox.DumpIndent(update, true)
		}
	}

}

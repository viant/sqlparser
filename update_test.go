package sqlparser_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlx/metadata/ast/parser"
	"github.com/viant/toolbox"
	"testing"
)

func TestParseUpdate(t *testing.T) {
	var testCases = []struct {
		description string
		SQL         string
		expect      string
	}{

		{
			description: "bq table select",
			SQL:         "UPDATE users SET name = 'Smith', last_access_time = ? WHERE id = 2",
			expect:      "UPDATE users SET name = 'Smith', last_access_time = ? WHERE id = 2",
		},
	}

	//for _, testCase := range testCases[len(testCases)-1:] {
	for _, testCase := range testCases {
		update, err := parser.ParseUpdate(testCase.SQL)
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

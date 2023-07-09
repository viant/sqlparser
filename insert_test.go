package sqlparser_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlparser"
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
		update, err := sqlparser.ParseInsert(testCase.SQL)
		if !assert.Nil(t, err) {
			fmt.Printf("%v\n", testCase.SQL)
			continue
		}

		actual := sqlparser.Stringify(update)
		if !assert.EqualValues(t, testCase.expect, actual) {
			data, _ := json.Marshal(update)
			fmt.Printf("%s\n", data)
		}
	}

}

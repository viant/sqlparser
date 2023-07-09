package sqlparser_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlparser"
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
		update, err := sqlparser.ParseUpdate(testCase.SQL)
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

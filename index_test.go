package sqlparser

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseCreateIndex(t *testing.T) {

	{

		var testCases = []struct {
			description string
			SQL         string
			expect      string
		}{

			{
				description: "basc create",
				SQL:         `CREATE UNIQUE INDEX MyIndex ON schema.table1(COL1, COL2);`,
				expect:      `CREATE UNIQUE INDEX MyIndex ON schema.table1(COL1, COL2);`,
			},
		}
		for _, testCase := range testCases {
			index, err := ParseCreateIndex(testCase.SQL)
			if !assert.Nil(t, err) {
				fmt.Printf("%v\n", testCase.SQL)
				continue
			}
			actual := Stringify(index)
			if !assert.EqualValues(t, testCase.expect, actual) {
				data, _ := json.Marshal(index)
				fmt.Printf("%s\n", data)
			}
		}
	}
}

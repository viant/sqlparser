package sqlparser_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/sqlparser"
	"testing"
)

func TestNewColumn(t *testing.T) {

	{

		var testCases = []struct {
			description string
			SQL         string
			expectLen   int
		}{

			{
				description: "quoted from expr",
				SQL:         `SELECT Id, Active, CAST(Lists AS LIST), TAG(Validation,'validate:"-" ') FROM Bar `,
				expectLen:   4,
			},
		}

		for _, testCase := range testCases {
			query, err := sqlparser.ParseQuery(testCase.SQL)
			if !assert.Nil(t, err, testCase.description) {
				continue
			}
			actual := sqlparser.NewColumns(query.List)
			assert.Equal(t, testCase.expectLen, len(actual), testCase.description)
		}
	}
}

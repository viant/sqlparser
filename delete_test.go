package sqlparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

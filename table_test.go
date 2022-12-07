package sqlparser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"testing"
)

func TestParseCreateTable(t *testing.T) {

	{

		var testCases = []struct {
			description string
			SQL         string
			expect      string
		}{

			{
				description: "basc create",
				SQL: `CREATE TABLE Bookstore2 (
 ISBN_NO varchar(15) PRIMARY KEY NOT NULL,
 SHORT_DESC varchar(100) ,
 AUTHOR varchar(40)  ,
 PUBLISHER varchar(40) ,
 ACTIVE bool DEFAULT 'true',
 PRICE float
);`,
				expect: `CREATE TABLE Bookstore2(
ISBN_NO varchar(15) PRIMARY KEY NOT NULL,
SHORT_DESC varchar(100),
AUTHOR varchar(40),
PUBLISHER varchar(40),
ACTIVE bool DEFAULT 'true',
PRICE float)`,
			},
		}
		for _, testCase := range testCases {
			table, err := ParseCreateTable(testCase.SQL)
			if !assert.Nil(t, err) {
				fmt.Printf("%v\n", testCase.SQL)
				continue
			}
			actual := Stringify(table)
			if !assert.EqualValues(t, testCase.expect, actual) {
				toolbox.DumpIndent(table, true)
			}
		}
	}
}

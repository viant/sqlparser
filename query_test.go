package sqlparser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"testing"
)

func TestParseSelect(t *testing.T) {

	{

		var testCases = []struct {
			description string
			SQL         string
			expect      string
		}{

			{
				description: "fun call",
				SQL:         `SELECT TOUPPER(name) AS Name FROM user u`,
				expect:      `SELECT TOUPPER(name) AS Name FROM user u`,
			},
			{
				description: "comments",
				SQL:         `SELECT user.* FROM user u -- extra comments `,
				expect:      `SELECT user.* FROM user u`,
			},

			{
				description: "expr with comments",
				SQL:         `SELECT user.* FROM (SELECT t.* FROM USER t  ) user /* {"Self":{"Holder":"Team", "Child":"ID", "Parent":"MGR_ID" }} */ `,
				expect:      `SELECT user.* FROM  (SELECT t.* FROM USER t  )  user /* {"Self":{"Holder":"Team", "Child":"ID", "Parent":"MGR_ID" }} */`,
			},

			{
				description: "bq table select",
				SQL:         "SELECT c1 /* comment */, c2 FROM `proj.dataset.table` t",
				expect:      "SELECT c1 /* comment */, c2 FROM `proj.dataset.table` t",
			},
			{
				description: "start with comments",
				SQL:         "SELECT t.* /* some comments */ FROM tableX t",
				expect:      "SELECT t.* /* some comments */ FROM tableX t",
			},

			{
				description: "bq table select",
				SQL:         "SELECT c1 AS a1 , c2 FROM `proj.dataset.table` t",
				expect:      "SELECT c1 AS a1, c2 FROM `proj.dataset.table` t",
			},

			{
				description: "except select",
				SQL:         "SELECT c1 /* comment */, c2 FROM x t",
				expect:      "SELECT c1 /* comment */, c2 FROM x t",
			},
			{
				description: "except select",
				SQL:         "SELECT * EXCEPT c1,c2 FROM x t",
				expect:      "SELECT * EXCEPT c1, c2 FROM x t",
			},

			{
				description: "except select",
				SQL:         "SELECT t1.* EXCEPT c1,c2, t2.* EXCEPT c3  FROM x t1 JOIN y AS t2 ON t1.ID=t2.ID",
				expect:      "SELECT t1.* EXCEPT c1, c2, t2.* EXCEPT c3 FROM x t1 JOIN y t2 ON t1.ID = t2.ID",
			},

			{
				description: "* select",
				SQL:         "SELECT * FROM x",
				expect:      "SELECT * FROM x",
			},
			{
				description: "* select with $",
				SQL:         "SELECT * FROM x WHERE id= $id",
				expect:      "SELECT * FROM x WHERE id = $id",
			},

			{
				description: "basic select",
				SQL:         "SELECT col1, t.col2, col3 AS col FROM x t",
				expect:      "SELECT col1, t.col2, col3 AS col FROM x t",
			},

			{
				description: "JOIN select",
				SQL:         "SELECT t.* FROM x1 t join x2 z ON t.ID = z.ID",
				expect:      "SELECT t.* FROM x1 t join x2 z ON t.ID = z.ID",
			},
			{
				description: "JOIN select",
				SQL:         "SELECT t.* FROM x1 t join x2 z ON t.ID = z.ID LEFT JOIN x3 y ON t.ID = x3.ID",
				expect:      "SELECT t.* FROM x1 t join x2 z ON t.ID = z.ID LEFT JOIN x3 y ON t.ID = x3.ID",
			},

			{
				description: "select with WHERE",
				SQL:         "SELECT t.* FROM x t WHERE 1=1 AND (x=2)",
				expect:      "SELECT t.* FROM x t WHERE 1 = 1 AND (x=2)",
			},

			{
				description: "func call select",
				SQL:         "SELECT COALESCE(t.PARENT_ID,0) AS PARENT, t.col2, col3 AS col FROM x t",
				expect:      "SELECT COALESCE(t.PARENT_ID,0) AS PARENT, t.col2, col3 AS col FROM x t",
			},
			{
				description: "exists select",
				SQL:         "SELECT 1 FROM x t WHERE col IN (1,2,3)",
				expect:      "SELECT 1 FROM x t WHERE col IN (1,2,3)",
			},

			{
				description: "unary operand select",
				SQL:         "SELECT NOT t.col FROM x t",
				expect:      "SELECT  NOT t.col FROM x t",
			},
			{
				description: "basic expr",
				SQL:         "SELECT col1 + col2 AS z, t.col2, col3 AS col FROM x t",
				expect:      "SELECT col1 + col2 AS z, t.col2, col3 AS col FROM x t",
			},
			{
				description: "between criteria select",
				SQL:         "SELECT c1 FROM table t WHERE a BETWEEN 1 AND 2",
				expect:      "SELECT c1 FROM table t WHERE a BETWEEN 1 AND 2",
			},
			{
				description: "between criteria select 2",
				SQL:         "SELECT c1 FROM table t WHERE a BETWEEN 1 AND 2 AND 1=1",
				expect:      "SELECT c1 FROM table t WHERE a BETWEEN 1 AND 2 AND 1 = 1",
			},
			{
				description: "join comments",
				SQL:         "SELECT * FROM tab1 t1 JOIN tab2 t2  /* my comment */  ON t1.ID = t2.ID ",
				expect:      "SELECT * FROM tab1 t1 JOIN tab2 t2 /* my comment */ ON t1.ID = t2.ID",
			},
			{
				description: "table comments",
				SQL:         "SELECT * FROM tab1 t1 /* my comment */ JOIN tab2 t2  ON t1.ID = t2.ID ",
				expect:      "SELECT * FROM tab1 t1 /* my comment */ JOIN tab2 t2 ON t1.ID = t2.ID",
			},
		}

		for _, testCase := range testCases {
			query, err := ParseQuery(testCase.SQL)
			if !assert.Nil(t, err) {
				fmt.Printf("%v\n", testCase.SQL)
				continue
			}

			actual := Stringify(query)
			if !assert.EqualValues(t, testCase.expect, actual) {
				toolbox.DumpIndent(query, true)
			}
		}
	}
}

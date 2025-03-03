package sqlparser

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"strings"
	"testing"
)

func TestBinaryWalk(t *testing.T) {
	query, _ := ParseQuery("SELECT * FROM t WHERE a = 1  AND (c IN(1,2))")
	binary, ok := query.Qualify.X.(*expr.Binary)
	if !assert.True(t, ok) {
		return
	}
	var actualColumns = make([]string, 0)
	binary.Walk(func(ident node.Node, values *expr.Values, operator, parentOperator string) error {
		actualColumns = append(actualColumns, Stringify(ident))
		return nil
	})
	assert.EqualValues(t, []string{"a", "c"}, actualColumns)

}

func TestParseSelect(t *testing.T) {

	{

		var testCases = []struct {
			description string
			SQL         string
			expect      string
			hasError    bool
			options     []Option
		}{

			{
				description: "except",
				SQL:         "SELECT main.* EXCEPT(Id), cast(main AS Record), cardinality(main, 'One') AS main FROM ta",
				expect:      "SELECT main.* EXCEPT Id, cast(main AS Record), cardinality(main, 'One') AS main FROM ta",
			},
			{
				description: "except group",
				SQL:         "SELECT main.* EXCEPT (Id,Name), cast(main AS Record), cardinality(main, 'One') AS main FROM ta",
				expect:      "SELECT main.* EXCEPT (Id, Name), cast(main AS Record), cardinality(main, 'One') AS main FROM ta",
			},

			{
				description: "criteria with  expr",
				SQL:         "SELECT Name FROM BAR WHERE ${predicate}",
				expect:      "SELECT Name FROM BAR WHERE ${predicate}",
			},
			{
				description: "quoted from expr",
				SQL:         "SELECT Name,Active FROM `/Records[Active = true]`",
				expect:      "SELECT Name, Active FROM `/Records[Active = true]`",
			},
			{
				description: "quoted from expr",
				SQL:         "SELECT * FROM $abc",
				expect:      "SELECT * FROM $abc",
			},
			{
				description: "with syntax",
				SQL: `WITH p AS (SELECT * FROM product), v AS (SELECT * FROM vendor)
				SELECT p.*, v.* FROM p JOIN v ON p.VENDOR_ID = v.ID`,
				expect: `SELECT p.*, v.* FROM (SELECT * FROM product) p JOIN (SELECT * FROM vendor) v ON p.VENDOR_ID = v.ID`,
			},

			{
				description: "group by",
				SQL:         `SELECT ID, SUM(amount) FROM product u GROUP BY 1`,
				expect:      `SELECT ID, SUM(amount) FROM product u GROUP BY 1`,
			},

			{
				description: "group by",
				SQL:         `SELECT ID, SUM(amount) FROM product u GROUP BY 1`,
				expect:      `SELECT ID, SUM(amount) FROM product u GROUP BY 1`,
			},
			{
				description: "group by, having",
				SQL:         `SELECT ID, SUM(amount) FROM product u GROUP BY 1 HAVING COUNT(DISTINCT zz) > 2`,
				expect:      `SELECT ID, SUM(amount) FROM product u GROUP BY 1 HAVING COUNT(DISTINCT zz) > 2`,
			},
			{
				description: "union all",
				SQL:         `SELECT user.* FROM user1 u UNION ALL SELECT user.* FROM user2 u`,
				expect:      `SELECT user.* FROM user1 u UNION ALL SELECT user.* FROM user2 u`,
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
				expect:      "SELECT * EXCEPT (c1, c2) FROM x t",
			},

			{
				description: "except select",
				SQL:         "SELECT t1.* EXCEPT c1,c2, t2.* EXCEPT c3  FROM x t1 JOIN y AS t2 ON t1.ID=t2.ID",
				expect:      "SELECT t1.* EXCEPT (c1, c2), t2.* EXCEPT c3 FROM x t1 JOIN y t2 ON t1.ID = t2.ID",
			},

			{
				description: "placeholder",
				SQL:         `SELECT ISBN, Name FROM Publication t WHERE (ISBN = ?) AND  1=1`,
				expect:      `SELECT ISBN, Name FROM Publication t WHERE (ISBN = ?) AND 1 = 1`,
			},

			{
				description: "error - extra coma",
				SQL:         `SELECT TOUPPER(name) AS Name, FROM user u`,
				hasError:    true,
			},
			{
				description: "fun call",
				SQL:         `SELECT TOUPPER(name) AS Name FROM user u`,
				expect:      `SELECT TOUPPER(name) AS Name FROM user u`,
			},

			{
				description: "cast call",
				SQL:         `SELECT CAST(name AS TEXT) AS Name FROM user u`,
				expect:      `SELECT CAST(name AS TEXT) AS Name FROM user u`,
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
				expect:      "SELECT * EXCEPT (c1, c2) FROM x t",
			},

			{
				description: "except select",
				SQL:         "SELECT t1.* EXCEPT c1,c2, t2.* EXCEPT c3  FROM x t1 JOIN y AS t2 ON t1.ID=t2.ID",
				expect:      "SELECT t1.* EXCEPT (c1, c2), t2.* EXCEPT c3 FROM x t1 JOIN y t2 ON t1.ID = t2.ID",
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

			{
				description: "criteria with additional expr",
				SQL:         "SELECT Name FROM BAR WHERE 1 = 1 ${expr}",
				expect:      "SELECT Name FROM BAR WHERE 1 = 1 ${expr}",
			},
			{
				description: "",
				expect:      `SELECT ID, Name, Price FROM dataset-name.abc.table-name v GROUP BY 1, 2`,
				SQL: `
                           select ID,
                                  Name,
                                  Price 
                           from dataset-name.abc.table-name v
                           group by 1, 2
                       `,
			},
			{
				description: "",
				SQL:         `select agegroup_id, name from (select id agegroup_id, name from CI_AGEGROUP) v #if($Has.AgeIncl)  where $criteria.In("v.agegroup_id", $AgeIncl) #end `,
				expect:      `SELECT agegroup_id, name FROM  (select id agegroup_id, name from CI_AGEGROUP)  v #if($Has.AgeIncl)  where $criteria.In("v.agegroup_id", $AgeIncl) #end`,
				options: []Option{
					WithErrorHandler(func(err error, cur *parsly.Cursor, destNode interface{}) error {
						fromNode, ok := destNode.(*query.From)
						if !ok {
							return err
						}
						input := string(cur.Input[cur.Pos:])
						if strings.HasPrefix(input, "#if") {
							index := strings.LastIndex(input, "#end")
							if index == -1 {
								return err
							}
							input = input[:index+4]
							fromNode.Unparsed = input + " "
							cur.Pos += index + 4
							return nil
						}
						return err
					}),
				},
			},

			{
				description: "",
				SQL:         `SELECT col1, col2 FROM table1 t, UNNEST(b) v`,
				expect:      `SELECT col1, col2 FROM table1 t , UNNEST(b) v`,
			},

			{
				description: "",
				SQL:         `SELECT               ID,NAME  FROM AAA               ORDER BY 2 DESC, 1 ASC`,
				expect:      `SELECT ID, NAME FROM AAA ORDER BY 2 DESC, 1 ASC`,
			},

			{
				description: "",
				SQL:         `SELECT col1, col2 FROM table1/tt t JOIN xx/e v ON v.ID=t.ID`,
				expect:      `SELECT col1, col2 FROM table1/tt t JOIN xx/e v ON v.ID = t.ID`,
			},
			{
				description: "",
				SQL:         `SELECT col1, col2 FROM table1[id=?].tt t JOIN xx/e v ON v.ID=t.ID`,
				expect:      `SELECT col1, col2 FROM table1[id=?].tt t JOIN xx/e v ON v.ID = t.ID`,
			},
		}

		for _, testCase := range testCases {
			//for _, testCase := range testCases {
			query, err := ParseQuery(testCase.SQL, testCase.options...)
			if testCase.hasError {
				assert.NotNilf(t, err, testCase.description)
				continue
			}
			if !assert.Nil(t, err) {
				fmt.Println(err)
				fmt.Printf("%v\n", testCase.SQL)
				continue
			}

			actual := strings.TrimSpace(Stringify(query))
			if !assert.EqualValues(t, testCase.expect, actual) {
				data, _ := json.Marshal(query)
				fmt.Printf("%s\n", data)
			}
		}
	}
}

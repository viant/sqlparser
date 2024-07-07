package sqlparser

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"testing"
)

func TestParseQualify(t *testing.T) {
	var testCases = []struct {
		expr           string
		expectEq       map[string]interface{}
		expectParentOp string
	}{
		{
			expr: "col1 = ?",
			expectEq: map[string]interface{}{
				"col1": "?",
			},
		},
		{
			expr: "col1 in(1.0,2.0,3.0)",
			expectEq: map[string]interface{}{
				"col1": []interface{}{1.0, 2.0, 3.0},
			},
		},
	}

	for _, testCase := range testCases[1:] {
		cursor := parsly.NewCursor("", []byte(testCase.expr), 0)
		qualify := &expr.Qualify{}
		err := ParseQualify(cursor, qualify)
		if !assert.Nil(t, err) {
			continue
		}
		actualEq := make(map[string]interface{})
		err = qualify.X.(*expr.Binary).Walk(func(ident node.Node, values *expr.Values, operator, parentOperator string) error {
			value := toValues(values)
			actualEq[Stringify(ident)] = value
			return nil
		})
		if !assert.Nil(t, err) {
			continue
		}
		assert.EqualValues(t, testCase.expectEq, actualEq)
	}

}

func toValues(values *expr.Values) interface{} {
	var value interface{}
	if len(values.X) == 1 {
		if values.X[0].Placeholder {
			value = "?"
		} else {
			value = values.X[0].Value
		}
	} else {
		aSlice := make([]interface{}, len(values.X))
		for i, v := range values.X {
			if v.Placeholder {
				aSlice[i] = "?"
			} else {
				aSlice[i] = v.Value
			}
		}
		value = aSlice
	}
	return value
}

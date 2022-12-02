package sqlparser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
)

//ParseQualify parses qualify expr
func ParseQualify(cursor *parsly.Cursor, qualify *expr.Qualify) error {
	binary := &expr.Binary{}
	err := parseBinaryExpr(cursor, binary)
	qualify.X = binary
	return err
}

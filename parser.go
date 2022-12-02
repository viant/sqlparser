package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	"github.com/viant/sqlparser/query"
)

func Parse(cursor *parsly.Cursor, dest interface{}) error {
	switch destination := dest.(type) {
	case *query.Select:
		return parseQuery(cursor, destination)
	default:
		return fmt.Errorf("not supported: %T", dest)
	}
}

package sqlparser

import (
	"fmt"
	"github.com/viant/parsly"
	del "github.com/viant/sqlparser/delete"
	"github.com/viant/sqlparser/insert"
	"github.com/viant/sqlparser/query"
	"github.com/viant/sqlparser/schema"
	"github.com/viant/sqlparser/update"
)

// Parse parses SQL into supplied destination
func Parse(cursor *parsly.Cursor, dest interface{}) error {
	switch destination := dest.(type) {
	case *query.Select:
		return parseQuery(cursor, destination)
	case *insert.Statement:
		return parseInsert(cursor, destination)
	case *update.Statement:
		return parseUpdate(cursor, destination)
	case *del.Statement:
		return parseDelete(cursor, destination)
	case *schema.Register:
		return parseRegisterType(cursor, destination)
	default:
		return fmt.Errorf("not supported: %T", dest)
	}
}

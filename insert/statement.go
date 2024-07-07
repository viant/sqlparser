package insert

import "github.com/viant/sqlparser/update"

// Statement represents an insert stmt
type Statement struct {
	Target               Target
	Alias                string
	Columns              []string
	Values               []*Value
	OnDuplicateKeyUpdate []*update.Item
}

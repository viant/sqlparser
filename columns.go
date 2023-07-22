package sqlparser

import (
	"github.com/viant/sqlparser/query"
	"strings"
)

// Columns represens column
type Columns []*Column

// Index indexes column by first non-empty alias or name or expr respectively
func (c Columns) Index() map[string]*Column {
	var result = make(map[string]*Column)
	for i, item := range c {
		result[item.Identity()] = c[i]
	}
	return result
}

// ByName indexes column by name
func (c Columns) ByName() map[string]*Column {
	var result = make(map[string]*Column)
	for i, item := range c {
		if item.Name != "" {
			result[item.Name] = c[i]
		}
	}
	return result
}

// ByLowerCasedName indexes column by lower cased name
func (c Columns) ByLowerCasedName() map[string]*Column {
	var result = make(map[string]*Column)
	for i, item := range c {
		if item.Name != "" {
			result[strings.ToLower(item.Name)] = c[i]
		}
	}
	return result
}

// Namespace returns namespace column
func (c Columns) Namespace(namespace string) Columns {
	var result = Columns{}
	for i, item := range c {
		if item.Namespace == namespace {
			result = append(result, c[i])
		}
	}
	return result
}

// StarExpr returns star expr
func (c Columns) StarExpr(namespace string) *Column {
	for _, item := range c {
		if item.Name == "*" && (item.Namespace == namespace || namespace == "") {
			return item
		}
	}
	return nil
}

func (c Columns) IsStarExpr() bool {
	if len(c) != 1 {
		return false
	}
	return strings.HasSuffix(c[0].Expression, "*")
}

// NewColumns creates a columns
func NewColumns(list query.List) Columns {
	var result Columns
	if len(list) == 0 {
		return result
	}
	for i := range list {
		column := NewColumn(list[i])
		result = append(result, column)
	}
	return result
}

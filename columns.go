package sqlparser

import "github.com/viant/sqlparser/query"

//Columns represens column
type Columns []*Column

//Index indexes column by alias or name or expr if not empty respectively
func (c Columns) Index() map[string]*Column {
	var result = make(map[string]*Column)
	for i, item := range c {
		if item.Alias != "" {
			result[item.Alias] = c[i]
			continue
		}
		if item.Name != "" {
			result[item.Name] = c[i]
			continue
		}
		if item.Expression != "" {
			result[item.Expression] = c[i]
		}
	}
	return result
}

//Namespace returns namespace column
func (c Columns) Namespace(namespace string) Columns {
	var result = Columns{}
	for i, item := range c {
		if item.Namespace == namespace {
			result = append(result, c[i])
		}
	}
	return result
}

//StarExpr returns star expr
func (c Columns) StarExpr(namespace string) *Column {
	for _, item := range c {
		if item.Name == "*" && (item.Namespace == namespace || namespace == "") {
			return item
		}
	}
	return nil
}

//NewColumns creates a columns
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

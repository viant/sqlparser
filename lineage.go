package sqlparser

import (
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
	"strings"
)

// ColumnOrigin is parser-proven direct table/column provenance.
type ColumnOrigin struct{ Table, Column string }
type Lineage struct{ Query *query.Select }

// ColumnLineage is immutable compiled projection provenance. Lookup handles
// proven wildcard sources without changing or walking an execution row type.
type ColumnLineage struct {
	wildcards         []lineageWildcard
	opaque            bool
	direct            map[string]ColumnOrigin
	blocked, excluded map[string]bool
	wildcard          *lineageSource
	root              string
}

func (c *ColumnLineage) Lookup(name string) (ColumnOrigin, bool) {
	if c == nil {
		return ColumnOrigin{}, false
	}
	key := strings.ToLower(name)
	if origin, ok := c.direct[key]; ok {
		return origin, true
	}
	if c.blocked[key] || c.excluded[key] || c.wildcard == nil {
		return ColumnOrigin{}, false
	}
	return c.wildcard.lookup(name)
}
func (c *ColumnLineage) RootTable() string {
	if c == nil {
		return ""
	}
	return c.root
}
func (l Lineage) Columns() map[string]ColumnOrigin {
	compiled := l.Compile()
	result := map[string]ColumnOrigin{}
	for key, value := range compiled.direct {
		result[key] = value
	}
	return result
}
func (l Lineage) RootTable() string       { return l.Compile().RootTable() }
func (l Lineage) Compile() *ColumnLineage { return l.compile(map[*query.Select]bool{}) }
func (l Lineage) compile(active map[*query.Select]bool) *ColumnLineage {
	result := &ColumnLineage{direct: map[string]ColumnOrigin{}, blocked: map[string]bool{}, excluded: map[string]bool{}}
	if l.Query == nil || l.Query.Union != nil || l.Query.WithRecursive || active[l.Query] {
		result.opaque = true
		return result
	}
	active[l.Query] = true
	defer delete(active, l.Query)
	root := l.source(l.Query.From.X, active)
	result.root = root.table
	alias := l.Query.From.Alias
	if alias == "" {
		alias = root.table
	}
	sources := map[string]lineageSource{strings.ToLower(alias): root}
	for _, join := range l.Query.Joins {
		if join == nil {
			continue
		}
		source := l.source(join.With, active)
		alias := join.Alias
		if alias == "" {
			alias = source.table
		}
		key := strings.ToLower(alias)
		if _, exists := sources[key]; exists {
			sources[key] = lineageSource{}
		} else {
			sources[key] = source
		}
	}
	sourceFor := func(namespace string) lineageSource {
		if namespace != "" {
			return sources[strings.ToLower(namespace)]
		}
		if len(sources) == 1 {
			for _, source := range sources {
				return source
			}
		}
		return lineageSource{}
	}
	stars := 0

	for _, item := range l.Query.List {
		if item == nil {
			continue
		}
		namespace := ""
		var star *expr.Star
		switch value := item.Expr.(type) {
		case *expr.Star:
			star = value
			switch qualifier := value.X.(type) {
			case *expr.Ident:
				if qualifier.Name != "*" {
					namespace = qualifier.Name
				}
			case *expr.Selector:
				namespace = qualifier.Name
			}
		case *expr.Selector:
			star, _ = value.X.(*expr.Star)
			namespace = value.Name
		}
		if star != nil {
			stars++
			source := sourceFor(namespace)
			result.wildcard = &source
			candidate := lineageWildcard{source: source, excluded: map[string]bool{}}
			for _, name := range star.Except {
				candidate.excluded[strings.ToLower(name)] = true
			}
			result.wildcards = append(result.wildcards, candidate)
			for _, name := range star.Except {
				result.excluded[strings.ToLower(name)] = true
			}
			continue
		}
		column := NewColumn(item)
		if column == nil {
			continue
		}
		key := strings.ToLower(column.Identity())
		if key == "" {
			continue
		}
		if column.Expression != "" || column.Name == "" {
			result.blocked[key] = true
			delete(result.direct, key)
			continue
		}
		origin, ok := sourceFor(column.Namespace).lookup(column.Name)
		if !ok {
			result.blocked[key] = true
			delete(result.direct, key)
			continue
		}
		if previous, exists := result.direct[key]; exists && previous != origin {
			result.blocked[key] = true
			delete(result.direct, key)
			continue
		}
		if !result.blocked[key] {
			result.direct[key] = origin
		}
	}
	for name := range result.direct {
		for _, wildcard := range result.wildcards {
			if wildcard.mayContain(name) {
				result.blocked[name] = true
				delete(result.direct, name)
				break
			}
		}
	}
	if stars != 1 {
		result.wildcard = nil
	}
	return result
}

type lineageSource struct {
	table      string
	projection *ColumnLineage
}

func (s lineageSource) lookup(name string) (ColumnOrigin, bool) {
	if s.table != "" {
		return ColumnOrigin{Table: s.table, Column: name}, true
	}
	if s.projection != nil {
		return s.projection.Lookup(name)
	}
	return ColumnOrigin{}, false
}
func (l Lineage) source(n node.Node, active map[*query.Select]bool) lineageSource {
	switch value := n.(type) {
	case *expr.Ident:
		if selected := l.Query.WithSelects.Select(value.Name); selected != nil {
			return lineageSource{projection: (Lineage{Query: selected.X}).compile(active)}
		}
		return lineageSource{table: value.Name}
	case *expr.Selector:
		return lineageSource{table: TableName(&query.Select{From: query.From{X: value}})}
	case *expr.Raw:
		if selected, ok := value.X.(*query.Select); ok {
			return lineageSource{projection: (Lineage{Query: selected}).compile(active)}
		}
	case *query.Select:
		return lineageSource{projection: (Lineage{Query: value}).compile(active)}
	case *expr.Parenthesis:
		if selected, ok := value.X.(*query.Select); ok {
			return lineageSource{projection: (Lineage{Query: selected}).compile(active)}
		}
	}
	return lineageSource{}
}

// An actual table (or unknown source) can contain any named column without
// schema discovery. A closed subquery projection can prove absence.
type lineageWildcard struct {
	source   lineageSource
	excluded map[string]bool
}

func (w lineageWildcard) mayContain(name string) bool {
	if w.excluded[name] {
		return false
	}
	if w.source.projection != nil {
		return w.source.projection.mayContain(name)
	}
	return true
}
func (c *ColumnLineage) mayContain(name string) bool {
	if c.opaque {
		return true
	}
	if _, ok := c.direct[name]; ok || c.blocked[name] {
		return true
	}
	for _, wildcard := range c.wildcards {
		if wildcard.mayContain(name) {
			return true
		}
	}
	return false
}

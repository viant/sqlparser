package sqlparser

import (
	"strings"

	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/query"
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
func (l Lineage) Compile() *ColumnLineage { return l.compile(nil, map[*query.Select]bool{}) }
func (l Lineage) compile(ctes map[string]*ColumnLineage, active map[*query.Select]bool) *ColumnLineage {
	result := &ColumnLineage{direct: map[string]ColumnOrigin{}, blocked: map[string]bool{}, excluded: map[string]bool{}}
	if l.Query == nil || l.Query.Union != nil || active[l.Query] {
		result.opaque = true
		return result
	}
	active[l.Query] = true
	defer delete(active, l.Query)
	ctes = l.cteScope(ctes, active)
	root := lineageOfSource(l.Query.From.X, ctes, active)
	result.root = root.table
	sources := map[string]lineageSource{lineageQualifier(l.Query.From.Alias, l.Query.From.X, root): root}
	for _, join := range l.Query.Joins {
		if join == nil {
			continue
		}
		source := lineageOfSource(join.With, ctes, active)
		key := lineageQualifier(join.Alias, join.With, source)
		if _, exists := sources[key]; exists {
			sources[key] = lineageSource{}
		} else {
			sources[key] = source
		}
	}
	sourceFor := func(namespace string) lineageSource {
		if namespace != "" {
			return sources[lineageName(namespace)]
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
		// NewColumn fills inferred metadata on its input item.
		itemCopy := *item
		column := NewColumn(&itemCopy)
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

func (l Lineage) cteScope(inherited map[string]*ColumnLineage, active map[*query.Select]bool) map[string]*ColumnLineage {
	if len(l.Query.WithSelects) == 0 {
		return inherited
	}
	scope := make(map[string]*ColumnLineage, len(inherited)+len(l.Query.WithSelects))
	for name, cte := range inherited {
		scope[name] = cte
	}
	opaque := &ColumnLineage{opaque: true}
	declared := map[string]int{}
	unresolved := map[string]bool{}
	for _, cte := range l.Query.WithSelects {
		name := lineageName(cte.Alias)
		declared[name]++
		if _, exists := scope[name]; !exists || l.Query.WithRecursive {
			// Recursive local names shadow inherited names throughout the WITH.
			// An unresolved local name must not be inferred as a physical table.
			scope[name] = opaque
		}
		if l.Query.WithRecursive {
			unresolved[name] = true
		}
	}
	for _, cte := range l.Query.WithSelects {
		name := lineageName(cte.Alias)
		if declared[name] > 1 {
			scope[name] = opaque
			continue
		}
		if l.Query.WithRecursive {
			self, pending := lineageCTEReferences(cte.X, name, unresolved)
			if pending {
				// A forward dependency may be part of a cycle. Keep this
				// name unresolved for later definitions that depend on it.
				scope[name] = opaque
				continue
			}
			delete(unresolved, name)
			if self {
				scope[name] = opaque
				continue
			}
		}
		scope[name] = (Lineage{Query: cte.X}).compile(scope, active)
	}
	return scope
}

// lineageCTEReferences finds self and unresolved sibling references in one
// scoped traversal, including sources outside the projection's provenance.
func lineageCTEReferences(statement *query.Select, ownName string, unresolved map[string]bool) (self, pending bool) {
	shadowed := map[string]int{}
	active := map[*query.Select]bool{}
	var check func(*query.Select)
	check = func(selected *query.Select) {
		if selected == nil || pending {
			return
		}
		if active[selected] {
			pending = true
			return
		}
		active[selected] = true
		defer delete(active, selected)

		var localNames []string
		shadow := func(alias string) {
			name := lineageName(alias)
			if unresolved[name] {
				shadowed[name]++
				localNames = append(localNames, name)
			}
		}
		if selected.WithRecursive {
			for _, cte := range selected.WithSelects {
				shadow(cte.Alias)
			}
		}
		for _, cte := range selected.WithSelects {
			check(cte.X)
			if !selected.WithRecursive {
				shadow(cte.Alias)
			}
		}

		Traverse(selected, func(n node.Node) bool {
			if pending {
				return false
			}
			var source node.Node
			switch actual := n.(type) {
			case *query.WithSelect:
				return false // Already checked in lexical declaration order.
			case *query.Select:
				if actual != selected {
					check(actual)
					return false
				}
			case *query.From:
				source = actual.X
			case *query.Join:
				source = actual.With
			}
			if ident, ok := source.(*expr.Ident); ok {
				name := lineageName(ident.Name)
				if unresolved[name] && shadowed[name] == 0 {
					if name == ownName {
						self = true
					} else {
						pending = true
					}
				}
			}
			return !pending
		})
		for _, name := range localNames {
			shadowed[name]--
		}
	}
	check(statement)
	return self, pending
}

func lineageQualifier(alias string, n node.Node, source lineageSource) string {
	if alias == "" {
		if named, ok := n.(*expr.Ident); ok {
			alias = named.Name
		} else {
			alias = source.table
		}
	}
	return lineageName(alias)
}

func lineageName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(trimEnclosure(name))
}

func lineageOfSource(n node.Node, ctes map[string]*ColumnLineage, active map[*query.Select]bool) lineageSource {
	switch value := n.(type) {
	case *expr.Ident:
		if cte, ok := ctes[lineageName(value.Name)]; ok {
			return lineageSource{projection: cte}
		}
		return lineageSource{table: value.Name}
	case *expr.Selector:
		return lineageSource{table: TableName(&query.Select{From: query.From{X: value}})}
	case *expr.Raw:
		if selected, ok := value.X.(*query.Select); ok {
			return lineageSource{projection: (Lineage{Query: selected}).compile(ctes, active)}
		}
	case *query.Select:
		return lineageSource{projection: (Lineage{Query: value}).compile(ctes, active)}
	case *expr.Parenthesis:
		if selected, ok := value.X.(*query.Select); ok {
			return lineageSource{projection: (Lineage{Query: selected}).compile(ctes, active)}
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

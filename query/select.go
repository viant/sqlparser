package query

import (
	"github.com/viant/sqlparser/expr"
)

type (
	Select struct {
		List        List
		From        From
		Joins       []*Join
		Qualify     *expr.Qualify
		GroupBy     List
		Having      *expr.Qualify
		OrderBy     List
		Window      *expr.Raw
		Limit       *expr.Literal
		Offset      *expr.Literal
		Kind        string
		Union       *Union
		WithSelects WithSelects
	}

	WithSelects []*WithSelect
	WithSelect  struct {
		Raw   string
		Alias string
		X     *Select
	}

	Union struct {
		Kind string
		Raw  string
		X    *Select
	}
)

func (w WithSelects) Select(alias string) *WithSelect {
	for _, candidate := range w {
		if candidate.Alias == alias {
			return candidate
		}
	}
	return nil
}

func (s *Select) IsNested() bool {
	if s.From.X == nil {
		return false
	}
	_, ok := s.From.X.(*expr.Raw)
	return ok
}

func (s *Select) NestedSelect() *Select {
	if s.From.X == nil {
		return nil
	}
	raw, ok := s.From.X.(*expr.Raw)
	if !ok {
		return nil
	}
	result, _ := raw.X.(*Select)
	return result
}

package sqlparser

import (
	"fmt"
	"strings"

	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/source"
)

// Cast describes a CAST operand and its authored type without interpreting the
// type as either a SQL dialect type or a host-language type.
type Cast struct {
	Operand string
	Type    string
}

// CastExpression returns nil for another function. It retains custom type
// spelling rather than applying NewColumn's legacy SQL type approximation.
// Both CAST(value AS type) and the declaration-compatible CAST(value,'type')
// form are accepted; consumers decide whether a call is executable or metadata.
func CastExpression(call *expr.Call) (*Cast, error) {
	if call == nil || !strings.EqualFold(strings.TrimSpace(Stringify(call.X)), "cast") {
		return nil, nil
	}
	raw := strings.TrimSpace(call.Raw)
	// ParseCallExpr historically retains the function prefix while query
	// parsing retains only its parenthesized source.
	if len(raw) >= 4 && strings.EqualFold(raw[:4], "cast") {
		raw = strings.TrimSpace(raw[4:])
	}
	group, end, ok := source.ReadGroupString(raw, 0, '(', ')')
	if !ok || strings.TrimSpace(raw[end:]) != "" {
		return nil, fmt.Errorf("CAST requires one complete parenthesized expression")
	}
	inside := group[1 : len(group)-1]
	result := &Cast{}
	if index := source.FindTopLevelKeyword(inside, "AS", 0); index >= 0 {
		result.Operand = strings.TrimSpace(inside[:index])
		result.Type = source.TrimQuote(strings.TrimSpace(inside[index+2:]))
	} else {
		args := source.SplitArgs(inside)
		if len(args) != 2 {
			return nil, fmt.Errorf("CAST requires operand and type")
		}
		result.Operand, result.Type = strings.TrimSpace(args[0]), source.TrimQuote(args[1])
	}
	if result.Operand == "" || result.Type == "" {
		return nil, fmt.Errorf("CAST requires non-empty operand and type")
	}
	return result, nil
}

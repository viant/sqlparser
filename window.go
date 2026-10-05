package sqlparser

import (
	"fmt"
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/source"
)

// matchWindowKeyword consumes whole words only, including intervening comments.
// A failed lookahead leaves the caller's cursor unchanged.
func matchWindowKeyword(cursor *parsly.Cursor, words ...string) bool {
	look := *cursor
	for _, word := range words {
		skipExpressionSpace(&look)
		start := look.Pos
		length := selectorMatcher.Match(&look)
		if length != len(word) || !strings.EqualFold(string(look.Input[start:start+length]), word) {
			return false
		}
		look.Pos += length
	}
	cursor.Pos = look.Pos
	return true
}

func parseWindowExpression(cursor *parsly.Cursor, call node.Node) (*expr.Window, error) {
	skipExpressionSpace(cursor)
	raw, end, ok := source.ReadGroupString(string(cursor.Input), cursor.Pos, '(', ')')
	if !ok {
		return nil, fmt.Errorf("OVER requires an inline parenthesized window specification")
	}
	inner := parsly.NewCursor(cursor.Path, []byte(raw[1:len(raw)-1]), cursor.Pos+1)
	result := &expr.Window{X: call}
	if matchWindowKeyword(inner, "PARTITION", "BY") {
		items, err := parseWindowItems(inner, false)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			result.PartitionBy = append(result.PartitionBy, item.X)
		}
	}
	if matchWindowKeyword(inner, "ORDER", "BY") {
		var err error
		result.OrderBy, err = parseWindowItems(inner, true)
		if err != nil {
			return nil, err
		}
	}
	for _, unit := range []string{"ROWS", "RANGE"} {
		if !matchWindowKeyword(inner, unit) {
			continue
		}
		frame := &expr.WindowFrame{Unit: unit}
		between := matchWindowKeyword(inner, "BETWEEN")
		var err error
		frame.Start, err = parseWindowBound(inner)
		if err != nil {
			return nil, err
		}
		if between {
			if !matchWindowKeyword(inner, "AND") {
				return nil, fmt.Errorf("window frame BETWEEN requires AND")
			}
			frame.End, err = parseWindowBound(inner)
			if err != nil {
				return nil, err
			}
		}
		if !validWindowFrame(frame) {
			return nil, fmt.Errorf("invalid window frame bounds")
		}
		result.Frame = frame
		break
	}
	skipExpressionSpace(inner)
	if inner.Pos != len(inner.Input) {
		return nil, fmt.Errorf("unsupported or malformed window specification at byte %d", inner.Pos)
	}
	cursor.Pos = end
	return result, nil
}

func parseWindowItems(cursor *parsly.Cursor, ordered bool) ([]*expr.WindowOrder, error) {
	var result []*expr.WindowOrder
	for {
		look := *cursor
		if matchWindowKeyword(&look, "ROWS") || matchWindowKeyword(&look, "RANGE") || matchWindowKeyword(&look, "PARTITION", "BY") {
			return nil, fmt.Errorf("expected window expression")
		}
		value, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		item := &expr.WindowOrder{X: value}
		if ordered {
			for _, direction := range []string{"ASC", "DESC"} {
				if matchWindowKeyword(cursor, direction) {
					item.Direction = direction
					break
				}
			}
		}
		result = append(result, item)
		skipExpressionSpace(cursor)
		if cursor.MatchOne(nextMatcher).Code != nextCode {
			return result, nil
		}
	}
}

func parseWindowBound(cursor *parsly.Cursor) (*expr.WindowBound, error) {
	if matchWindowKeyword(cursor, "CURRENT", "ROW") {
		return &expr.WindowBound{Kind: "CURRENT ROW"}, nil
	}
	bound := &expr.WindowBound{}
	if matchWindowKeyword(cursor, "UNBOUNDED") {
		bound.Kind = "UNBOUNDED "
	} else {
		value, err := expectOperand(cursor)
		if err != nil {
			return nil, err
		}
		switch actual := value.(type) {
		case *expr.Placeholder:
		case *expr.Literal:
			if actual.Value == "" || strings.Trim(actual.Value, "0123456789") != "" || actual.Kind == "string" {
				return nil, fmt.Errorf("window frame offset must be a nonnegative integer or parameter")
			}
		default:
			return nil, fmt.Errorf("window frame offset must be a nonnegative integer or parameter")
		}
		bound.X = value
	}
	for _, direction := range []string{"PRECEDING", "FOLLOWING"} {
		if matchWindowKeyword(cursor, direction) {
			bound.Kind += direction
			return bound, nil
		}
	}
	return nil, fmt.Errorf("window frame bound requires PRECEDING or FOLLOWING")
}

func validWindowFrame(frame *expr.WindowFrame) bool {
	rank := map[string]int{"UNBOUNDED PRECEDING": 0, "PRECEDING": 1, "CURRENT ROW": 2, "FOLLOWING": 3, "UNBOUNDED FOLLOWING": 4}
	start := rank[frame.Start.Kind]
	end := 2 // Without BETWEEN, the implicit end is CURRENT ROW.
	if frame.End != nil {
		end = rank[frame.End.Kind]
	}
	return start != 4 && end != 0 && start <= end
}

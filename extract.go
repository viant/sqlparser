package sqlparser

import (
	"fmt"
	"strings"

	"github.com/viant/parsly"
	"github.com/viant/sqlparser/expr"
	"github.com/viant/sqlparser/node"
	"github.com/viant/sqlparser/source"
)

// The date part is syntax, not a column reference. The source and optional
// time-zone expressions remain executable AST children for visitors.
func parseExtractArguments(parent *parsly.Cursor, raw string, pos int) ([]node.Node, error) {
	inside := raw[1 : len(raw)-1]
	index := source.FindTopLevelKeyword(inside, "FROM", 0)
	if index < 0 {
		return nil, fmt.Errorf("EXTRACT requires date part FROM expression")
	}
	part, err := parseExtractPart(inside[:index])
	if err != nil {
		return nil, err
	}
	cursor := parsly.NewCursor(parent.Path, []byte(inside[index+4:]), pos+1+index+4)
	cursor.OnError = parent.OnError
	operand, err := expectExpression(cursor)
	if err != nil {
		return nil, err
	}
	skipExpressionSpace(cursor)
	if cursor.Pos != len(cursor.Input) {
		var words []string
		for _, keyword := range []string{"AT", "TIME", "ZONE"} {
			word := readExtractWord(cursor)
			if !strings.EqualFold(word, keyword) {
				return nil, fmt.Errorf("EXTRACT requires AT TIME ZONE after its source expression")
			}
			words = append(words, word)
		}
		zone, err := expectExpression(cursor)
		if err != nil {
			return nil, err
		}
		operand = &expr.Binary{X: operand, Op: strings.Join(words, " "), Y: zone}
		skipExpressionSpace(cursor)
	}
	if operand == nil || cursor.Pos != len(cursor.Input) {
		return nil, fmt.Errorf("EXTRACT requires one complete expression after FROM")
	}
	return []node.Node{&expr.Binary{X: &expr.Raw{Raw: part}, Op: inside[index : index+4], Y: operand}}, nil
}

func parseExtractPart(raw string) (string, error) {
	cursor := parsly.NewCursor("EXTRACT date part", []byte(raw), 0)
	part := readExtractWord(cursor)
	if part == "" {
		return "", fmt.Errorf("EXTRACT requires a date part")
	}
	skipExpressionSpace(cursor)
	if strings.EqualFold(part, "WEEK") && cursor.Pos < len(cursor.Input) && cursor.Input[cursor.Pos] == '(' {
		cursor.Pos++
		weekday := readExtractWord(cursor)
		switch strings.ToUpper(weekday) {
		case "SUNDAY", "MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY":
		default:
			return "", fmt.Errorf("unsupported EXTRACT weekday %q", weekday)
		}
		skipExpressionSpace(cursor)
		if cursor.Pos >= len(cursor.Input) || cursor.Input[cursor.Pos] != ')' {
			return "", fmt.Errorf("EXTRACT WEEK requires one parenthesized weekday")
		}
		cursor.Pos++
		part += "(" + weekday + ")"
		skipExpressionSpace(cursor)
	}
	if cursor.Pos != len(cursor.Input) {
		return "", fmt.Errorf("unsupported EXTRACT date part %q", raw)
	}
	return part, nil
}

// Date parts, weekdays and clause keywords are unquoted words; comments may
// separate them but cannot join fragments of a single word.
func readExtractWord(cursor *parsly.Cursor) string {
	skipExpressionSpace(cursor)
	start := cursor.Pos
	for cursor.Pos < len(cursor.Input) {
		c := cursor.Input[cursor.Pos]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || cursor.Pos > start && c >= '0' && c <= '9') {
			break
		}
		cursor.Pos++
	}
	return string(cursor.Input[start:cursor.Pos])
}

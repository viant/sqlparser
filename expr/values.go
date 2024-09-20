package expr

import (
	"fmt"
	"github.com/viant/sqlparser/node"
	"strconv"
	"strings"
)

type (
	//Value represents predicate value
	Value struct {
		Placeholder bool
		Raw         string
		Value       interface{}
		Kind        string
	}
	Values struct {
		Idx int
		X   []Value
	}
)

func (v *Value) AsInt() (int, bool) {
	ret, ok := v.Value.(int)
	if ok {
		return ret, true
	}
	f, ok := v.Value.(float64)
	if ok {
		return int(f), true
	}
	return 0, false
}

func NewValue(raw string) (*Value, error) {
	ret := &Value{Raw: raw}
	if strings.HasPrefix(raw, "'") {
		ret.Value = strings.Trim(raw, "'")
		ret.Kind = "string"
	} else {
		switch strings.ToLower(raw) {
		case "null":
			ret.Value = nil
			ret.Kind = "null"
		case "true":
			ret.Value = true
			ret.Kind = "bool"
		case "false":
			ret.Value = false
			ret.Kind = "bool"
		}
		if strings.Contains(raw, ".") {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, err
			}
			ret.Value = v
			ret.Kind = "numeric"

		} else {
			v, err := strconv.Atoi(raw)
			if err != nil {
				return nil, err
			}
			ret.Value = v
			ret.Kind = "int"
		}
	}
	return ret, nil
}

func (v *Values) append(value ...Value) {
	v.X = append(v.X, value...)
}

// Values returns values
func (v *Values) Values(placeholderProvider func(idx int) interface{}) []interface{} {
	var result = make([]interface{}, len(v.X))

	for i, item := range v.X {
		if item.Placeholder {
			result[i] = placeholderProvider(v.Idx)
			v.Idx++
			continue
		}
		result[i] = item.Value
	}
	return result
}

// NewValues creates predicate values
func NewValues(n node.Node) (*Values, error) {
	var values = Values{}
	switch actual := n.(type) {
	case *Placeholder:
		values.append(Value{Placeholder: true})
		return &values, nil
	case *Binary:
		switch actualY := actual.Y.(type) {
		case *Binary:
			return NewValues(actual.X)
		case *Parenthesis:
			return NewValues(actualY.X)
		}

		return NewValues(actual.Y)
	case *Literal:
		switch actual.Kind {
		case "int":
			v, err := strconv.Atoi(actual.Value)
			if err != nil {
				return nil, err
			}
			values.append(Value{Value: v, Kind: actual.Kind})
			return &values, nil
		case "null":
			values.append(Value{Value: nil, Kind: actual.Kind})
			return &values, nil
		case "string":
			values.append(Value{Value: strings.Trim(actual.Value, "'"), Kind: actual.Kind})
			return &values, nil
		case "numeric":
			v, err := strconv.ParseFloat(actual.Value, 64)
			if err != nil {
				return nil, err
			}
			values.append(Value{Value: v, Kind: actual.Kind})
			return &values, nil
		}
	case *Parenthesis:
		list, ok := actual.X.([]node.Node)
		if ok {
			for _, item := range list {
				v, err := NewValues(item)
				if err != nil {
					return nil, err
				}
				values.append(v.X...)
			}
			return &values, nil
		}
	case *Range:
		aRange, ok := n.(*Range)
		if ok {
			vMin, err := NewValues(aRange.Min)
			if err != nil {
				return nil, err
			}

			vMax, err := NewValues(aRange.Max)
			if err != nil {
				return nil, err
			}

			values.append(vMin.X...)
			values.append(vMax.X...)
			return &values, nil
		}
	case []node.Node:
		for _, item := range actual {
			if aValue, ok := item.(*Literal); ok {
				v, _ := NewValue(aValue.Value)
				values.append(*v)
			}
		}
		return &values, nil
	}

	return nil, fmt.Errorf("unsupported value node: %T", n)
}

package matcher

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/parsly"
	"testing"
)

func TestSelector_Match(t *testing.T) {
	useCases := []struct {
		description string
		input       []byte
		matched     bool
		isTable     bool
	}{
		{
			description: "t[id=?].collection matches",
			input:       []byte("tw[@id=?].collection test"),
			matched:     true,
			isTable:     true,
		},
		{
			description: "t.abc matches",
			input:       []byte("t.abc test"),
			matched:     true,
			isTable:     false,
		},
		{
			description: "unicode doesn't match",
			input:       []byte("日本語 test"),
			matched:     false,
			isTable:     false,
		},
		{
			description: "underscore matches",
			input:       []byte("ABc_test"),
			matched:     true,
			isTable:     false,
		},
		{
			description: "- doesn't match",
			input:       []byte("ABc-test"),
			matched:     true,
			isTable:     false,
		},
		{
			description: "beginning number doesn't match",
			input:       []byte("9ABctest"),
			matched:     false,
			isTable:     false,
		},
	}

	for _, useCase := range useCases {
		matcher := NewSelector(useCase.isTable)
		matched := matcher.Match(parsly.NewCursor("", useCase.input, 0))
		assert.Equal(t, useCase.matched, matched > 0, useCase.description)
	}
}

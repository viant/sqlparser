package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/parsly"
)

func TestDollarQuoteLiteral(t *testing.T) {
	for _, raw := range []string{"$$$$", "$$x$$", "$tag$[)--$tag$", "$_tag1$x$_tag1$"} {
		cursor := parsly.NewCursor("", []byte(raw+" FROM src"), 0)
		literal, err := ParseLiteral(cursor)
		require.NoError(t, err)
		require.Equal(t, raw, literal.Value)
		require.Equal(t, len(raw), cursor.Pos)
	}
	for _, raw := range []string{"$$unclosed", "$tag$x$TAG$"} {
		_, err := ParseLiteral(parsly.NewCursor("", []byte(raw), 0))
		require.Error(t, err)
	}
}

func TestDollarQuotePlaceholderAllocations(t *testing.T) {
	for _, parameter := range []string{"$name", "$1", "${name}", "$name.Member"} {
		cursor := parsly.NewCursor("", []byte(parameter+", "+strings.Repeat("x", 1<<16)), 0)
		allocs := testing.AllocsPerRun(100, func() {
			cursor.Pos = 0
			literal, err := TryParseLiteral(cursor)
			if literal != nil || err != nil {
				t.Fatalf("placeholder parsed as literal: %v, %v", literal, err)
			}
		})
		require.Zero(t, allocs, parameter)
	}
}

func BenchmarkDollarQuoteDetection(b *testing.B) {
	for _, prefix := range []string{"$name", "$$x$$"} {
		b.Run(prefix, func(b *testing.B) {
			cursor := parsly.NewCursor("", []byte(prefix+", "+strings.Repeat("x", 1<<16)), 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cursor.Pos = 0
				if _, err := TryParseLiteral(cursor); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

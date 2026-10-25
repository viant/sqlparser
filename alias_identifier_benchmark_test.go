package sqlparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/viant/parsly"
)

var aliasIdentifierBenchmarkLength int

// Keep the token constant while changing unrelated input length. The original
// matcher copies the whole input; MatchBytes should only visit the token.
func BenchmarkAliasIdentifierMatch(b *testing.B) {
	identifier := aliasIdentifier{}
	for _, token := range []struct{ name, text string }{
		{"ascii", "report_id"},
		{"unicode", "_値٢"},
		{"quoted", `"report""id"`},
		{"invalid", "\xff"},
	} {
		for _, length := range []int{64, 4096, 65536} {
			input := []byte("SELECT " + token.text + " " + strings.Repeat(" ", length))[:length]
			cursor := parsly.NewCursor("", input, 0)
			cursor.Pos = len("SELECT ")
			for _, implementation := range []struct {
				name  string
				match func(*parsly.Cursor) int
			}{
				{"string", identifier.matchStringDeprecated},
				{"bytes", identifier.MatchBytes},
			} {
				b.Run(fmt.Sprintf("%s/input_%d/%s", token.name, length, implementation.name), func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						aliasIdentifierBenchmarkLength = implementation.match(cursor)
					}
				})
			}
		}
	}
}

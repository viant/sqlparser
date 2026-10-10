package source

import (
	"strings"
	"testing"
)

func BenchmarkCodeScannerTraversal(b *testing.B) {
	for _, fragment := range []string{"?,", "? /* ? */ '?' ? -- ?\n"} {
		b.Run(fragment, func(b *testing.B) {
			SQL := strings.Repeat(fragment, 32766)
			b.SetBytes(int64(len(SQL)))
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				scanner := NewCodeScanner(SQL, 0)
				for _, ok := scanner.Next(); ok; _, ok = scanner.Next() {
				}
			}
		})
	}
}

func BenchmarkFindCodeToken(b *testing.B) {
	SQL := strings.Repeat("SELECT id, name FROM records WHERE name != 'literal' /* protected text */;\n", 24)
	for _, suffix := range []string{"", " '$View.ParentJoinOn'", " $View.ParentJoinOn('AND', 'id')"} {
		name := "absent"
		if suffix != "" {
			name = "protected"
			if suffix[1] != '\'' {
				name = "code"
			}
		}
		b.Run(name, func(b *testing.B) {
			text := SQL + suffix
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				FindCodeToken(text, "$View.", 0)
			}
		})
	}
}

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

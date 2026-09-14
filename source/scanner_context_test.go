package source

import (
	"fmt"
	"strings"
	"testing"
)

func TestCodeScannerPreviousSignificant(t *testing.T) {
	for _, SQL := range []string{
		"", "?", " \t\r\n\f\v?", "-- only comment", "/* only comment */",
		"SELECT /* outer /* ? */ inner */ ? -- ?\n, ?",
		"'a''?' /* ? */ ?", "'a\\'?' ?", "\"a\"\"?\" ?", "`a``?` ?", "[a]]?] ?",
		"$$ ? $$ /* x */ ?", "$tag$ ? $tag$ -- x\n?", "$1 ? $unclosed$ ?",
		"? /* unclosed", "'unclosed ?", "? -- a\n /* b */ -- c\n ?",
		"SELECT :nested.Value /* x */ ? ?| ?& @?",
	} {
		t.Run(SQL, func(t *testing.T) {
			// Starting inside any quote/comment must retain the context that a
			// scan from byte zero would provide, without exposing protected bytes.
			for start := -1; start <= len(SQL)+1; start++ {
				scanner := NewCodeScanner(SQL, start)
				for position, ok := scanner.Next(); ok; position, ok = scanner.Next() {
					if ProtectionAt(SQL, position) != "" {
						t.Fatalf("exposed protected byte %d at start %d", position, start)
					}
					previous := position - 1
					kind := ""
					for previous >= 0 {
						if IsWhitespace(SQL[previous]) {
							previous--
							continue
						}
						begin, _, protection := ProtectedRangeAt(SQL, previous)
						if strings.Contains(protection, "comment") {
							previous = begin - 1
							continue
						}
						kind = protection
						break
					}
					got, gotKind := scanner.PreviousSignificant()
					if got != previous || gotKind != kind {
						t.Fatalf("start=%d position=%d previous=(%d,%q) want=(%d,%q)", start, position, got, gotKind, previous, kind)
					}
				}
			}
		})
	}
}

func TestCodeScannerContextBoundedAllocations(t *testing.T) {
	SQL := strings.Repeat("? /* ? */ '?' ? -- ?\n", 32766)
	allocs := testing.AllocsPerRun(1, func() {
		scanner := NewCodeScanner(SQL, 0)
		count := 0
		for position, ok := scanner.Next(); ok; position, ok = scanner.Next() {
			previous, _ := scanner.PreviousSignificant()
			if previous >= position {
				t.Fatal("context did not precede current position")
			}
			if SQL[position] == '?' {
				count++
			}
		}
		if count != 2*32766 {
			t.Fatalf("executable question marks=%d", count)
		}
	})
	// The constructor may allocate the scanner once, but traversal and context
	// must not allocate per byte, protected region or question mark.
	if allocs > 1 {
		t.Fatalf("scanner/context allocations=%g want<=1", allocs)
	}
}

func BenchmarkCodeScannerContextScaling(b *testing.B) {
	for _, count := range []int{512, 2048, 8192, 32766} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			SQL := strings.Repeat("? /* ? */ '?' ? -- ?\n", count)
			b.ReportAllocs()
			b.SetBytes(int64(len(SQL)))
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				scanner := NewCodeScanner(SQL, 0)
				for position, ok := scanner.Next(); ok; position, ok = scanner.Next() {
					previous, _ := scanner.PreviousSignificant()
					if previous >= position {
						b.Fatal("context did not precede current position")
					}
				}
			}
		})
	}
}

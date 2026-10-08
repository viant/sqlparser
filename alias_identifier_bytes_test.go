package sqlparser

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/viant/parsly"
)

func assertAliasIdentifierEquivalent(t *testing.T, input []byte, position int) {
	t.Helper()
	before := append([]byte(nil), input...)
	cursor := parsly.NewCursor("", input, 0)
	cursor.Pos = position
	originalCursor := *cursor
	identifier := aliasIdentifier{}
	want := identifier.matchStringDeprecated(cursor)
	if !reflect.DeepEqual(*cursor, originalCursor) {
		t.Fatal("string matcher changed the cursor")
	}
	got := identifier.MatchBytes(cursor)
	if got != want {
		t.Fatalf("input=%q position=%d: bytes length=%d, string length=%d", input, position, got, want)
	}
	if !reflect.DeepEqual(*cursor, originalCursor) || !bytes.Equal(input, before) {
		t.Fatal("bytes matcher changed the cursor or input")
	}
}

func aliasIdentifierCompatibilityInputs() []string {
	return []string{
		"", "name", "_", "_name2", "name$2", "éclair", "_値2", "from値",
		"from$value", "NOT", "NOTLIKE", "NOT LIKE", "NOT /* gap */ LIKE",
		"GLOB", "GLOB_pattern", "WHERE", "where値", "value.name", "name!",
		"1name", "$name", "name-with-dash", "name\x00", "name\xff", "\xffname",
		"\uFFFD", "\u0301name", "a\u0301", "a١", "١name", "name\u00a0FROM",
		"'literal'", `""`, "``", "[]", `"name"`, "`name`", "[name]",
		`"a""b"`, "`a``b`", "[a]]b]", `"name"suffix`, `"unclosed`, "[unclosed",
		"`unclosed", "\"a\x00b\"", "\"a\xffb\"", "[a\xffb]", "[]]", "[[]",
		"name -- tail\r\nFROM table", "name/* comment */", "x\tname", "x\n値",
	}
}

func TestAliasIdentifierBytesEquivalent(t *testing.T) {
	for _, input := range aliasIdentifierCompatibilityInputs() {
		// Include positions inside UTF-8 sequences and beyond the end. Both
		// matchers must preserve the original behavior, not add stricter syntax.
		for position := 0; position <= len(input)+1; position++ {
			assertAliasIdentifierEquivalent(t, []byte(input), position)
		}
	}
}

func TestAliasIdentifierBytesEquivalentRandomInputs(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for sample := 0; sample < 2000; sample++ {
		input := make([]byte, random.Intn(64))
		if _, err := random.Read(input); err != nil {
			t.Fatal(err)
		}
		assertAliasIdentifierEquivalent(t, input, random.Intn(len(input)+2))
	}
}

func TestAliasIdentifierBytesDoesNotAllocate(t *testing.T) {
	identifier := aliasIdentifier{}
	for _, input := range aliasIdentifierCompatibilityInputs() {
		cursor := parsly.NewCursor("", []byte(input), 0)
		if allocations := testing.AllocsPerRun(100, func() {
			aliasIdentifierBenchmarkLength = identifier.MatchBytes(cursor)
		}); allocations != 0 {
			t.Fatalf("input=%q: got %v allocations, want zero", input, allocations)
		}
	}
}

func FuzzAliasIdentifierBytesEquivalent(f *testing.F) {
	for _, input := range aliasIdentifierCompatibilityInputs() {
		f.Add([]byte(input), uint32(0))
		f.Add([]byte(input), uint32(len(input)))
	}
	f.Fuzz(func(t *testing.T, input []byte, offset uint32) {
		if len(input) > 4096 {
			t.Skip()
		}
		position := int(offset % uint32(len(input)+2))
		assertAliasIdentifierEquivalent(t, input, position)
	})
}

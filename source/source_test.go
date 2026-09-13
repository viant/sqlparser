package source

import (
	"reflect"
	"strings"
	"testing"
)

func TestProtectedRegions(t *testing.T) {
	for _, protected := range []string{"'$TOKEN'", `"$TOKEN"`, "`$TOKEN`", "[$TOKEN]", "$$ $TOKEN $$", "$tag$ $TOKEN $tag$", "/* $TOKEN */", "/* outer /* $TOKEN */ end */", "-- $TOKEN\n", "'O''Brien $TOKEN'", "'O\\'Brien $TOKEN'"} {
		t.Run(protected, func(t *testing.T) {
			source := "SELECT " + protected + " $TOKEN"
			expected := strings.LastIndex(source, "$TOKEN")
			if actual := Token("$TOKEN").Find(source, 0); actual != expected {
				t.Fatalf("found %d wanted %d", actual, expected)
			}
			if actual := Token("$TOKEN").ReplaceAll(source, "?"); actual != source[:expected]+"?" {
				t.Fatalf("replace %s", actual)
			}
			if ProtectionAt(source, strings.Index(source, "$TOKEN")) == "" {
				t.Fatal("missing protection")
			}
			if ProtectionAt(source, expected) != "" {
				t.Fatal("protected executable token")
			}
		})
	}
}

func TestSourceStructure(t *testing.T) {
	if _, _, ok := ReadGroupString("[a]]", 0, '[', ']'); ok {
		t.Fatal("unterminated bracket quote accepted")
	}
	for _, tt := range []struct {
		name, source string
		parts        []string
	}{
		{"nested", "id, coalesce(name, 'a,b'), (SELECT max(id) FROM x)", []string{"id", "coalesce(name, 'a,b')", "(SELECT max(id) FROM x)"}},
		{"quoted", "'O''Brien,a', [a,b], $$x,y$$, `a,b`", []string{"'O''Brien,a'", "[a,b]", "$$x,y$$", "`a,b`"}},
		{"comments", "id /* , */, name -- ,\n, last", []string{"id /* , */", "name -- ,", "last"}},
		{"braces", "{ID,Name}, fn(a,b)", []string{"{ID,Name}", "fn(a,b)"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if actual := SplitTopLevelCSV(tt.source); !reflect.DeepEqual(actual, tt.parts) {
				t.Fatalf("%#v != %#v", actual, tt.parts)
			}
		})
	}
	for _, tt := range []struct {
		source      string
		open, close byte
		want        string
	}{
		{"(a, fn(')')) tail", '(', ')', "(a, fn(')'))"},
		{"{a,{b,c}} tail", '{', '}', "{a,{b,c}}"},
		{"[a]]b] tail", '[', ']', "[a]]b]"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			group, end, ok := ReadGroupString(tt.source, 0, tt.open, tt.close)
			if !ok || group != tt.want || end != len(tt.want) {
				t.Fatalf("group %q end %d ok %v", group, end, ok)
			}
		})
	}
}

func TestClauseBoundaries(t *testing.T) {
	for _, tt := range []struct {
		source, keyword string
		want            int
	}{
		{"SELECT 'where' FROM t WHERE id=1", "where", 22},
		{"WITH t AS (SELECT 1 FROM x WHERE id=2) SELECT * FROM t ORDER BY id", "select", 39},
		{"SELECT * FROM t WHERE id IN (SELECT id FROM x ORDER BY id) ORDER BY id", "order by", 59},
		{"SELECT somewhere FROM t", "where", -1},
	} {
		t.Run(tt.source, func(t *testing.T) {
			if actual := FindTopLevelKeyword(tt.source, tt.keyword, 0); actual != tt.want {
				t.Fatalf("got %d expected %d", actual, tt.want)
			}
		})
	}
	for _, tail := range []string{"GROUP BY id", "HAVING count(*)>1", "ORDER BY id", "LIMIT 1", "OFFSET 2", "UNION SELECT id FROM other", ";"} {
		source := "SELECT id FROM t WHERE id IN (SELECT id FROM x LIMIT 2) " + tail
		if got := CriteriaBoundary(source); got != strings.Index(source, tail) {
			t.Fatalf("boundary %d in %s", got, source)
		}
	}
	for _, tail := range []string{"ORDER/*x*/BY id", "GROUP--x\nBY id", "FOR/*x*/UPDATE"} {
		source := "SELECT id FROM t " + tail
		if got := CriteriaBoundary(source); got != len("SELECT id FROM t ") {
			t.Fatalf("comment-separated boundary %d in %s", got, source)
		}
	}
	if got := FindTopLevelKeyword(") ORDER BY x", "order by", 0); got != 2 {
		t.Fatalf("unmatched close hides clause: %d", got)
	}
	for _, tail := range []string{" -- trailing comment", " /* trailing comment */", " \n"} {
		if got := CriteriaBoundary("SELECT id FROM t" + tail); got != len("SELECT id FROM t") {
			t.Fatalf("trailing boundary %d", got)
		}
	}
}

func TestProjectionAliases(t *testing.T) {
	for _, tt := range []struct{ source, core, alias string }{
		{"COUNT(*) AS total", "COUNT(*)", "total"},
		{`COUNT(*) "Total Count"`, "COUNT(*)", `"Total Count"`},
		{"coalesce(a, 'as') total", "coalesce(a, 'as')", "total"},
		{"a + b", "a + b", ""},
		{"CASE WHEN a=1 THEN 2 ELSE 3 END", "CASE WHEN a=1 THEN 2 ELSE 3 END", ""},
		{"name COLLATE nocase", "name COLLATE nocase", ""},
	} {
		t.Run(tt.source, func(t *testing.T) {
			core, alias := SplitTopLevelAlias(tt.source)
			if core != tt.core || alias != tt.alias {
				t.Fatalf("core %q alias %q", core, alias)
			}
		})
	}
}

func TestTokenBoundaryAndOffsets(t *testing.T) {
	source := "'$VIEW' $VIEW_LONG x$VIEW $VIEW"
	if got := Token("$VIEW").Find(source, 0); got != strings.LastIndex(source, "$VIEW") {
		t.Fatalf("token at %d", got)
	}
	if got := Token("$VIEW").Find(source, 2); got != strings.LastIndex(source, "$VIEW") {
		t.Fatalf("offset exposes quote: %d", got)
	}
}

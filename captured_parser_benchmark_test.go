package sqlparser

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/sqlparser/query"
)

type capturedIdentifierParserInput struct {
	SQL       string `json:"-"`
	SQLFile   string `json:"sql_file"`
	SQLSHA256 string `json:"sql_sha256"`
	Caller    string `json:"caller"`
	Options   int    `json:"options"`
	Failed    bool   `json:"failed"`
}

var capturedIdentifierParsedQuery *query.Select

func loadCapturedIdentifierParserInputs(tb testing.TB, mode string) []capturedIdentifierParserInput {
	tb.Helper()
	root := filepath.Join("testdata", "identifier_benchmark", mode)
	data, err := os.ReadFile(filepath.Join(root, "parser_calls.json"))
	if err != nil {
		tb.Fatal(err)
	}
	var inputs []capturedIdentifierParserInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		tb.Fatal(err)
	}
	for i := range inputs {
		input := &inputs[i]
		if input.SQLFile == "" || !filepath.IsLocal(input.SQLFile) {
			tb.Fatal("captured SQL must be a relative path within the fixture")
		}
		sql, err := os.ReadFile(filepath.Join(root, input.SQLFile))
		if err != nil {
			tb.Fatal(err)
		}
		// SQL files can have a Git-friendly final LF; the SHA identifies the
		// exact bytes originally passed to ParseQuery, not reformatted SQL.
		if fmt.Sprintf("%x", sha256.Sum256(sql)) != input.SQLSHA256 && len(sql) > 0 && sql[len(sql)-1] == '\n' {
			sql = sql[:len(sql)-1]
		}
		if fmt.Sprintf("%x", sha256.Sum256(sql)) != input.SQLSHA256 {
			tb.Fatalf("captured SQL checksum changed: %s", input.SQLFile)
		}
		input.SQL = string(sql)
		if input.SQL == "" || input.Options != 0 {
			tb.Fatal("unsupported captured SQL/options; extend the replay before using it")
		}
		parsed, err := ParseQuery(input.SQL)
		if (err != nil) != input.Failed || (!input.Failed && parsed == nil) {
			tb.Fatalf("captured parser outcome changed (%s): %v", input.Caller, err)
		}
	}
	if len(inputs) != 9 {
		tb.Fatalf("expected nine captured ParseQuery calls per reader, got %d", len(inputs))
	}
	return inputs
}

// The A/B runner compares these snapshots across separate test processes, with
// only the Match delegation changed. No global parser state is switched here.
func TestCapturedAliasIdentifierParser(t *testing.T) {
	type outcome struct {
		Name      string
		AST       *query.Select
		Rendered  string
		ErrorText string
	}
	var outcomes []outcome
	appendOutcome := func(name, sql string) {
		parsed, err := ParseQuery(sql)
		result := outcome{Name: name, AST: parsed}
		if err != nil {
			result.ErrorText = err.Error()
		} else {
			result.Rendered = Stringify(parsed)
		}
		outcomes = append(outcomes, result)
	}
	for _, mode := range []string{"legacy", "compact"} {
		for i, input := range loadCapturedIdentifierParserInputs(t, mode) {
			appendOutcome(fmt.Sprintf("%s/%d/%s", mode, i, filepath.Base(input.SQLFile)), input.SQL)
		}
	}
	for i, sql := range []string{
		`SELECT value AS "a""b" FROM records AS "r""s"`,
		"SELECT value AS `a``b` FROM records AS `r``s`",
		`SELECT value AS [a]]b] FROM records AS [r]]s]`,
		`SELECT value AS éclair FROM records AS _値2`,
		`SELECT value AS from_records FROM records AS where_values`,
		`SELECT value NOT LIKE 'a%' FROM records AS t`,
		`SELECT value NOT GLOB 'a*' FROM records AS t`,
		"SELECT value AS name -- comment\r\nFROM records AS t",
		`SELECT value AS "unclosed FROM records`,
		`SELECT value AS name! FROM records`,
		"SELECT value AS name\xff FROM records",
		"SELECT value AS \"a\x00b\" FROM records",
	} {
		appendOutcome(fmt.Sprintf("compatibility/%d", i), sql)
	}
	data, err := json.MarshalIndent(outcomes, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Only an explicitly requested diagnostic run writes outside the checkout.
	if output := os.Getenv("SQLPARSER_BENCH_AST_OUTPUT"); output != "" {
		if err := os.WriteFile(output, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("recorded %d parser outcomes (AST, rendered SQL and errors)", len(outcomes))
}

// Each operation replays the nine ParseQuery calls for one warmed reader. This
// isolates the parser: no database, Datly, file I/O or tracing in the timer.
func BenchmarkCapturedParseQueryReaderInputs(b *testing.B) {
	for _, mode := range []string{"legacy", "compact"} {
		inputs := loadCapturedIdentifierParserInputs(b, mode)
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, input := range inputs {
					capturedIdentifierParsedQuery, _ = ParseQuery(input.SQL)
				}
			}
			b.ReportMetric(float64(len(inputs)), "parses/read")
		})
	}
}

func BenchmarkCapturedParseQueryUniqueInputs(b *testing.B) {
	for _, mode := range []string{"legacy", "compact"} {
		seen := map[string]bool{}
		for _, input := range loadCapturedIdentifierParserInputs(b, mode) {
			if seen[input.SQLSHA256] {
				continue
			}
			seen[input.SQLSHA256] = true
			name := fmt.Sprintf("%s/%s/bytes_%d", mode, strings.TrimSuffix(filepath.Base(input.SQLFile), ".sql"), len(input.SQL))
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					capturedIdentifierParsedQuery, _ = ParseQuery(input.SQL)
				}
			})
		}
	}
}

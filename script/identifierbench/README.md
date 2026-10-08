# Alias identifier matcher: original vs byte-scanning implementation

The checkout retains both implementations:

- `alias_identifier.go`: `matchStringDeprecated` contains the original `Match`
  body, retained only as a reference for differential tests and A/B benchmarks.
- `alias_identifier_bytes.go`: `MatchBytes` recognizes the same token directly
  in `cursor.Input`, without copying the entire SQL into a string or building
  a decoded quoted name that the matcher would discard.

**The normal parser uses `MatchBytes`.** `matchStringDeprecated` is deprecated
because it copies the entire SQL input into a string on every identifier match.
The comparison runner does not change the public parser API,
`tableIdentifierParser.part()`, Agently, or module files.
Both implementations use byte positions. Unicode letters/digits, doubled quote
escapes, brackets, invalid input and the existing single-quote behavior are
covered by differential tests. No `unsafe` conversions are used.

## Compare the complete parser

Run from the `sqlparser` checkout:

```sh
go run ./script/identifierbench -variant both -benchtime=1s -count=3
```

The runner executes the entire test suite with each implementation, compares
30 parser outcomes (serialized AST, rendered SQL and errors), then benchmarks
both variants. For the `string` baseline, a temporary **Go build overlay** changes
only the `Match` delegation to `matchStringDeprecated`. The `bytes` variant uses
the normal `MatchBytes` delegation unchanged. The overlay does not edit the
checkout and is removed when the command finishes. No application environment
variable is needed.

To run the variants separately:

```sh
go run ./script/identifierbench -variant string -benchtime=1s -count=3
go run ./script/identifierbench -variant bytes -benchtime=1s -count=3
```

Use `-go /absolute/path/to/go` if child test processes need a specific toolchain.
Keep Go version, dependency versions and `GOMAXPROCS` identical for both runs.

The default benchmark replays the exact nine `ParseQuery` calls captured from
each warmed Datly reader (`legacy` and `compact`). One `ns/op` is the sum of
those nine parses, **not** the duration of a database read or conversation
deletion. SQL, caller metadata and SHA-256 checksums are stored under
`testdata/identifier_benchmark`; five distinct SQL inputs cover both workloads.
Fixtures originate from Agently Core's cleanup parser benchmark. Their captured
parser options are the defaults; fixture loading and validation are outside
the benchmark timer. No database or network is used by these benchmarks.
Captured SQL retains its original whitespace (including trailing spaces) so
that replayed input and SHA-256 checksums remain identical to the capture.

Fixture names describe the parser input rather than its checksum:

- `legacy/parser_sql/conversation_reader.sql`: the general conversation
  reader query, before narrowing the selected fields, with named parameters.
- `legacy/parser_sql/conversation_reader_projected.sql`: that query wrapped
  in a five-column projection, with positional `?` parameters.
- `compact/parser_sql/conversation_reader.sql`: the smaller, dedicated graph
  reader query.
- `legacy/parser_sql/projection_source.sql` and
  `compact/parser_sql/projection_source.sql`: small synthetic fragments used
  by Datly while determining projection columns, not actual database reads.

`parser_calls.json` preserves the captured call order, repeated inputs and
caller names. Its `sql_file` entries reference these readable filenames;
`sql_sha256` independently verifies that the captured SQL bytes are unchanged.

To compare individual SQL inputs instead:

```sh
go run ./script/identifierbench -variant both \
  -bench '^BenchmarkCapturedParseQueryUniqueInputs$' -count=3
```

## Compare the matcher alone and fuzz compatibility

```sh
go test . -run '^TestAliasIdentifierBytes' -count=1
go test . -run '^$' -bench '^BenchmarkAliasIdentifierMatch$' -benchmem
go test . -run '^$' -fuzz '^FuzzAliasIdentifierBytesEquivalent$' \
  -fuzztime=10s -parallel=2
```

The matcher benchmark keeps the token fixed and varies the surrounding SQL
length (64 B, 4 KiB, 64 KiB). It runs `matchStringDeprecated` and `MatchBytes` directly;
no overlay is required. The zero-allocation assertion applies to `MatchBytes`,
not to the complete parser. Fuzzing also checks that neither matcher changes
the cursor or its input.

## Measurement on 2026-10-07

Go 1.25.8, macOS amd64, Intel i9-9980HK, GOMAXPROCS=16, three 1-second samples
per workload/variant; medians from one A/B run:

| Nine-parse workload | Original time | Byte variant time | Original B/op | Byte variant B/op |
| --- | ---: | ---: | ---: | ---: |
| Legacy reader | 18.081 ms | 9.541 ms | 16,344,101 | 2,231,643 |
| Compact reader | 0.510 ms | 0.403 ms | 187,325 | 112,576 |

The legacy workload was about 1.9x faster and allocated about 86% fewer bytes.
`B/op` is cumulative allocation per operation, not peak resident memory.
The full suite passed for both variants and their parser snapshots matched.
Differential fuzzing passed 103,899 executions in a 10-second run. The byte
matcher also reported zero allocations for ASCII, Unicode and quoted tokens
at both 4 KiB and 64 KiB input sizes.
These results isolate the parser and do not predict the same speedup for
end-to-end cleanup. At the time of this measurement, the runtime still selected
the original string matcher. The current runtime default is `MatchBytes`.

Static checks: `go vet ./...` reports three pre-existing unkeyed composite
literals in `call_case_test.go` (`TestCallCasePublicLayout`). That unrelated
test is unchanged. `go vet -composites=false ./...` passes.

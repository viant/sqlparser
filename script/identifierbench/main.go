// identifierbench compares the deprecated and byte-scanning alias matchers without
// changing checked-in sources, module files or the public parser API.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const byteDelegation = "return identifier.MatchBytes(cursor)"
const stringDelegation = "return identifier.matchStringDeprecated(cursor)"

func deprecatedMatcherOverlay(source []byte) ([]byte, error) {
	if bytes.Count(source, []byte(byteDelegation)) != 1 {
		return nil, fmt.Errorf("expected exactly one byte Match delegation; inspect alias_identifier.go before switching")
	}
	return bytes.Replace(source, []byte(byteDelegation), []byte(stringDelegation), 1), nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	variant := flag.String("variant", "both", "matcher: string, bytes or both")
	goBinary := flag.String("go", "go", "Go executable for child test processes")
	bench := flag.String("bench", "^BenchmarkCapturedParseQueryReaderInputs$", "benchmark regular expression")
	benchTime := flag.String("benchtime", "1s", "time/iterations per benchmark sample")
	count := flag.Int("count", 3, "benchmark samples per variant")
	flag.Parse()
	if flag.NArg() != 0 || *count < 1 {
		return fmt.Errorf("expected flags only and count >= 1")
	}
	var variants []string
	switch *variant {
	case "string", "bytes":
		variants = []string{*variant}
	case "both":
		variants = []string{"string", "bytes"}
	default:
		return fmt.Errorf("variant must be string, bytes or both")
	}

	data, err := exec.Command(*goBinary, "list", "-mod=readonly", "-m", "-json").Output()
	if err != nil {
		return fmt.Errorf("locate module (run from the sqlparser checkout): %w", err)
	}
	var module struct{ Path, Dir string }
	if err := json.Unmarshal(data, &module); err != nil {
		return err
	}
	if module.Path != "github.com/viant/sqlparser" || module.Dir == "" {
		return fmt.Errorf("run this tool from the github.com/viant/sqlparser module")
	}
	temporary, err := os.MkdirTemp("", "sqlparser-identifier-ab-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary) // Only the directory created by this invocation.

	sourcePath := filepath.Join(module.Dir, "alias_identifier.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	modified, err := deprecatedMatcherOverlay(source)
	if err != nil {
		return err
	}
	replacement := filepath.Join(temporary, "alias_identifier.go")
	if err := os.WriteFile(replacement, modified, 0600); err != nil {
		return err
	}
	overlay, err := json.Marshal(struct{ Replace map[string]string }{
		Replace: map[string]string{sourcePath: replacement},
	})
	if err != nil {
		return err
	}
	overlayPath := filepath.Join(temporary, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0600); err != nil {
		return err
	}

	invoke := func(variant, snapshot string, arguments ...string) error {
		args := []string{"test", "-mod=readonly"}
		if variant == "string" {
			args = append(args, "-overlay="+overlayPath)
		}
		args = append(args, arguments...)
		command := exec.Command(*goBinary, args...)
		command.Dir, command.Stdout, command.Stderr = module.Dir, os.Stdout, os.Stderr
		// Avoid inheriting a caller's diagnostic snapshot path.
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "SQLPARSER_BENCH_AST_OUTPUT=") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, "SQLPARSER_BENCH_AST_OUTPUT="+snapshot)
		if err := command.Run(); err != nil {
			return fmt.Errorf("%s variant: %w", variant, err)
		}
		return nil
	}
	for _, current := range variants {
		fmt.Printf("\n=== %s matcher: complete test suite ===\n", current)
		if err := invoke(current, filepath.Join(temporary, current+".json"), "-count=1", "./..."); err != nil {
			return err
		}
	}
	if len(variants) == 2 {
		original, err := os.ReadFile(filepath.Join(temporary, "string.json"))
		if err != nil {
			return err
		}
		optimized, err := os.ReadFile(filepath.Join(temporary, "bytes.json"))
		if err != nil {
			return err
		}
		if !bytes.Equal(original, optimized) {
			return fmt.Errorf("A/B parser AST, rendered SQL or error snapshots differ")
		}
		fmt.Println("\nA/B parser snapshots are identical (AST, rendered SQL, errors).")
	}
	for _, current := range variants {
		fmt.Printf("\n=== %s matcher: benchmarks ===\n", current)
		if err := invoke(current, "", ".", "-run=^$", "-bench="+*bench, "-benchmem", "-benchtime="+*benchTime, fmt.Sprintf("-count=%d", *count)); err != nil {
			return err
		}
	}
	return nil
}

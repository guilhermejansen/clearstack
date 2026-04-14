package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dustin/go-humanize"

	"github.com/guilhermejansen/clearstack/internal/ui/tui"
)

// writeJSON encodes v as pretty-printed JSON to w.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// humanBytes wraps go-humanize for consistent formatting.
func humanBytes(n int64) string {
	if n < 0 {
		return "0 B"
	}
	return humanize.IBytes(uint64(n))
}

// printfln is a tiny fmt.Fprintln shim that ignores write errors because the
// CLI is not responsible for recovering from a broken stdout.
func printfln(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

// runTUI launches the interactive clearstack terminal UI.
//
// Paths come from positional args; when none are provided we fall back to
// the user's home directory, which mirrors the CLI's scan/clean default.
func runTUI(_ io.Writer, args []string) error {
	eng, j, err := loadConfigAndEngine(buildEngineOptions{WithCleaner: true})
	if err != nil {
		return err
	}
	if j != nil {
		defer func() { _ = j.Close() }()
	}
	roots := args
	if len(roots) == 0 {
		roots = []string{defaultScanRoot()}
	}
	return tui.Run(tui.Options{Engine: eng, Roots: roots})
}

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dustin/go-humanize"
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

// runTUIPlaceholder is invoked when the user runs `clearstack` with no
// subcommand. Sprint 4 replaces this with a real Bubble Tea program.
func runTUIPlaceholder(w io.Writer) error {
	printfln(w, "clearstack — interactive TUI arrives in Sprint 4.")
	printfln(w, "For now try: clearstack scan --json | clearstack clean --dry-run | clearstack doctor")
	return nil
}

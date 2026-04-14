package main

import (
	"context"
	"fmt"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/engine"
	"github.com/guilhermejansen/clearstack/internal/platform"
)

var scanFlags struct {
	Categories []string
	OlderThan  string
	Workers    int
	NoSize     bool
}

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan directories for cleanable developer artifacts",
		Long: `scan walks the given paths (or ~/Developer by default) and reports
every detected cleanup candidate alongside its category and size.

scan never modifies anything. For cleanup, use 'clearstack clean'.`,
		Example: `  clearstack scan ~/Developer
  clearstack scan ~/code --categories=node_modules,next_cache --json
  clearstack scan ~/code --older-than=30d`,
		RunE: runScan,
	}
	f := cmd.Flags()
	f.StringSliceVar(&scanFlags.Categories, "categories", nil, "comma-separated category ids (default: all)")
	f.StringVar(&scanFlags.OlderThan, "older-than", "", "override dormancy threshold (e.g., 30d)")
	f.IntVar(&scanFlags.Workers, "workers", 0, "scanner worker count (default: NumCPU)")
	f.BoolVar(&scanFlags.NoSize, "no-size", false, "skip size calculation (faster, size=0)")
	return cmd
}

func runScan(cmd *cobra.Command, args []string) error {
	roots := args
	if len(roots) == 0 {
		roots = []string{defaultScanRoot()}
	}

	eng, _, err := loadConfigAndEngine(buildEngineOptions{
		Categories: toCategorySlice(scanFlags.Categories),
	})
	if err != nil {
		return err
	}
	if scanFlags.OlderThan != "" {
		d, err := parseDuration(scanFlags.OlderThan)
		if err != nil {
			return err
		}
		eng.Dormancy.MinAge = d
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	matches, errc := eng.Scan(ctx, roots)
	var collected []detectors.Match
	for m := range matches {
		collected = append(collected, m)
	}
	if err := <-errc; err != nil {
		printfln(cmd.ErrOrStderr(), "scan: %v", err)
	}
	if !scanFlags.NoSize {
		eng.SizeInPlace(ctx, collected)
	}

	return emitScanResults(cmd, collected, roots)
}

func emitScanResults(cmd *cobra.Command, matches []detectors.Match, roots []string) error {
	if globalFlags.JSON {
		return writeJSON(cmd.OutOrStdout(), map[string]any{
			"roots":   roots,
			"count":   len(matches),
			"matches": matches,
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].SizeBytes > matches[j].SizeBytes
	})
	var total int64
	for _, m := range matches {
		total += m.SizeBytes
		printfln(cmd.OutOrStdout(), "%-22s %10s  %s",
			m.Category, humanBytes(m.SizeBytes), m.Path)
	}
	printfln(cmd.OutOrStdout(), "\n%d matches  · total %s", len(matches), humanBytes(total))
	return nil
}

func defaultScanRoot() string {
	h := platform.Home()
	if h == "" {
		return "."
	}
	return h
}

func toCategorySlice(s []string) []detectors.Category {
	out := make([]detectors.Category, 0, len(s))
	for _, v := range s {
		if v == "" {
			continue
		}
		out = append(out, detectors.Category(v))
	}
	return out
}

func parseDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	if len(raw) > 0 && raw[len(raw)-1] == 'd' {
		d, err := time.ParseDuration(raw[:len(raw)-1] + "h")
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", raw, err)
		}
		return d * 24, nil
	}
	return time.ParseDuration(raw)
}

// ensure engine is imported even when unused in sub-files.
var _ = context.TODO
var _ engine.Config

package main

import (
	"bufio"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

var cleanFlags struct {
	Categories []string
	OlderThan  string
	Hard       bool
}

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [paths...]",
		Short: "Clean cleanable developer artifacts found in the given paths",
		Long: `clean walks the given paths, applies all matching detectors, and
executes each category's default strategy (trash, hard delete, or native
command). Results are appended to the operation journal and can be undone
via 'clearstack undo'.

Use --dry-run to preview without touching anything. This is the default for
first-time runs until the user confirms.`,
		Example: `  clearstack clean ~/Developer --dry-run
  clearstack clean ~/Developer --categories=node_modules,next_cache --yes
  clearstack clean ~/Developer --categories=go_mod_cache --yes`,
		RunE: runClean,
	}
	f := cmd.Flags()
	f.StringSliceVar(&cleanFlags.Categories, "categories", nil, "category ids to clean")
	f.StringVar(&cleanFlags.OlderThan, "older-than", "", "override dormancy threshold (e.g., 30d)")
	f.BoolVar(&cleanFlags.Hard, "hard", false, "bypass trash and delete directly (irreversible)")
	return cmd
}

func runClean(cmd *cobra.Command, args []string) error {
	roots := args
	if len(roots) == 0 {
		roots = []string{defaultScanRoot()}
	}

	eng, j, err := loadConfigAndEngine(buildEngineOptions{
		Categories:  toCategorySlice(cleanFlags.Categories),
		WithCleaner: true,
	})
	if err != nil {
		return err
	}
	if j != nil {
		defer func() { _ = j.Close() }()
	}
	if cleanFlags.OlderThan != "" {
		d, err := parseDuration(cleanFlags.OlderThan)
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
	eng.SizeInPlace(ctx, collected)

	if len(collected) == 0 {
		printfln(cmd.OutOrStdout(), "nothing to clean.")
		return nil
	}

	opts := detectors.CleanOptions{DryRun: globalFlags.DryRun}
	if cleanFlags.Hard {
		opts.Override = detectors.StrategyHardDelete
	}

	// Safety gate: force dry-run on the very first clean invocation unless
	// the user explicitly passes --yes.
	if !globalFlags.DryRun && !globalFlags.Yes {
		if !confirmInteractive(cmd, collected) {
			printfln(cmd.OutOrStdout(), "aborted.")
			return nil
		}
	}

	summary := eng.Clean(ctx, collected, opts)
	if globalFlags.JSON {
		return writeJSON(cmd.OutOrStdout(), summary)
	}
	printfln(cmd.OutOrStdout(), "%s %d/%d items, freed %s",
		verb(summary.DryRun), summary.Succeeded, summary.Attempted, humanBytes(summary.BytesFreed))
	if summary.Failed > 0 {
		printfln(cmd.ErrOrStderr(), "%d failures — inspect the journal", summary.Failed)
	}
	return nil
}

func verb(dry bool) string {
	if dry {
		return "would clean"
	}
	return "cleaned"
}

func confirmInteractive(cmd *cobra.Command, matches []detectors.Match) bool {
	var total int64
	for _, m := range matches {
		total += m.SizeBytes
	}
	printfln(cmd.OutOrStdout(), "about to clean %d items totalling %s.",
		len(matches), humanBytes(total))
	printfln(cmd.OutOrStdout(), "re-run with --yes to skip this prompt, or --dry-run to preview.")
	printfln(cmd.OutOrStdout(), "continue? (y/N)")

	in := bufio.NewReader(cmd.InOrStdin())
	line, err := in.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(line), "y")
}

// silence unused import when building without logic that references fmt.
var _ = fmt.Sprint

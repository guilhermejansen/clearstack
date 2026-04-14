package main

import (
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/journal"
	"github.com/guilhermejansen/clearstack/internal/platform"
)

var undoFlags struct {
	Last int
}

func newUndoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "List or restore previous cleanup operations from the journal",
		Long: `undo reads the operation journal and shows the most recent entries,
optionally restoring the last N items. Restoration is only possible for
entries recorded with the trash strategy (the default).

This sprint implements the listing side; the actual restore wiring is
completed alongside the TUI in Sprint 4 once we can surface it visually.`,
		RunE: runUndo,
	}
	cmd.Flags().IntVar(&undoFlags.Last, "last", 10, "number of recent entries to show")
	return cmd
}

func runUndo(cmd *cobra.Command, _ []string) error {
	path := filepath.Join(platform.StateDir(), "operations.jsonl")
	entries, err := journal.Read(path)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].At.After(entries[j].At)
	})
	if undoFlags.Last > 0 && len(entries) > undoFlags.Last {
		entries = entries[:undoFlags.Last]
	}

	if globalFlags.JSON {
		return writeJSON(cmd.OutOrStdout(), entries)
	}
	if len(entries) == 0 {
		printfln(cmd.OutOrStdout(), "no operations recorded (%s)", path)
		return nil
	}
	for _, e := range entries {
		status := "ok"
		if e.Err != "" {
			status = "ERR"
		} else if e.DryRun {
			status = "dry"
		}
		printfln(cmd.OutOrStdout(), "%s  [%s] %-16s %-12s %s", e.At.Format("2006-01-02 15:04:05"), status, e.Category, e.Strategy, e.OriginalPath)
	}
	return nil
}

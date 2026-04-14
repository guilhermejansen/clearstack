package main

import (
	"os/signal"
	"sort"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/engine"
)

var analyzeFlags struct {
	Top int
}

func newAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [paths...]",
		Short: "Summarize disk usage of cleanable categories",
		Long: `analyze walks the given paths and aggregates every match by category,
producing a top-N report of where the reclaimable space actually lives.`,
		RunE: runAnalyze,
	}
	cmd.Flags().IntVar(&analyzeFlags.Top, "top", 20, "number of top-sized matches to list")
	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	roots := args
	if len(roots) == 0 {
		roots = []string{defaultScanRoot()}
	}
	eng, _, err := loadConfigAndEngine(buildEngineOptions{})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	matches, errc := eng.Scan(ctx, roots)
	var all []detectors.Match
	for m := range matches {
		all = append(all, m)
	}
	if err := <-errc; err != nil {
		printfln(cmd.ErrOrStderr(), "scan: %v", err)
	}
	eng.SizeInPlace(ctx, all)

	byCat := map[detectors.Category]engine.CategoryStat{}
	var total int64
	for _, m := range all {
		s := byCat[m.Category]
		s.Matches++
		s.Bytes += m.SizeBytes
		byCat[m.Category] = s
		total += m.SizeBytes
	}

	if globalFlags.JSON {
		return writeJSON(cmd.OutOrStdout(), map[string]any{
			"roots":    roots,
			"total":    total,
			"by_cat":   byCat,
			"top":      topN(all, analyzeFlags.Top),
			"match_ct": len(all),
		})
	}
	printfln(cmd.OutOrStdout(), "\nBy category:")
	var keys []detectors.Category
	for k := range byCat {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return byCat[keys[i]].Bytes > byCat[keys[j]].Bytes })
	for _, k := range keys {
		s := byCat[k]
		printfln(cmd.OutOrStdout(), "  %-22s %10s  (%d items)", k, humanBytes(s.Bytes), s.Matches)
	}
	printfln(cmd.OutOrStdout(), "\nTop %d individual matches:", analyzeFlags.Top)
	top := topN(all, analyzeFlags.Top)
	for _, m := range top {
		printfln(cmd.OutOrStdout(), "  %-22s %10s  %s", m.Category, humanBytes(m.SizeBytes), m.Path)
	}
	printfln(cmd.OutOrStdout(), "\nTotal: %s across %d items", humanBytes(total), len(all))
	return nil
}

func topN(matches []detectors.Match, n int) []detectors.Match {
	sorted := make([]detectors.Match, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SizeBytes > sorted[j].SizeBytes })
	if n > 0 && len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

// Package engine orchestrates scanning, classification, sizing, and cleaning
// of developer disk artifacts. It wires detectors, a parallel scanner, an
// on-demand sizer, dormancy filtering, a cleaner, and a journal.
package engine

import (
	"time"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

// ScanOptions configures a single scan invocation.
type ScanOptions struct {
	// Roots are the directories to walk. Empty means "use configured default".
	Roots []string
	// Categories filters detectors. Empty means "all enabled detectors".
	Categories []detectors.Category
	// FollowSymlinks is intentionally hardwired to false at the scanner layer
	// for safety; this field is reserved for future explicit opt-in.
	FollowSymlinks bool
	// NumWorkers bounds scanner concurrency. Zero means runtime.NumCPU().
	NumWorkers int
	// Dormancy filters matches whose mtime is newer than the threshold.
	// Zero disables dormancy filtering regardless of detector opinion.
	Dormancy time.Duration
	// CheckGit augments dormancy with `git log -1 --format=%ct` when true.
	CheckGit bool
}

// ScanSummary aggregates results of a completed scan.
type ScanSummary struct {
	StartedAt   time.Time
	CompletedAt time.Time
	Matches     int
	TotalBytes  int64
	Errors      int
	Roots       []string
	ByCategory  map[detectors.Category]CategoryStat
}

// CategoryStat reports per-category aggregates.
type CategoryStat struct {
	Matches int
	Bytes   int64
}

// CleanSummary aggregates results of a cleaning run.
type CleanSummary struct {
	StartedAt   time.Time
	CompletedAt time.Time
	Attempted   int
	Succeeded   int
	Failed      int
	DryRun      bool
	BytesFreed  int64
	Errors      []error
}

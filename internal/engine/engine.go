package engine

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/journal"
)

// Engine is the high-level orchestrator that wires a scanner, sizer,
// dormancy policy, cleaner, and detector registry into a single API.
//
// Most callers should build an Engine with NewEngine once per process and
// then invoke Scan() / Clean() as needed.
type Engine struct {
	Registry *detectors.Registry
	Safety   *Safety
	Scanner  *Scanner
	Sizer    *Sizer
	Dormancy DormancyPolicy
	Cleaner  *Cleaner
	Journal  *journal.Journal
}

// Config bundles the inputs needed to construct an Engine.
type Config struct {
	Registry   *detectors.Registry
	Safety     *Safety
	Dormancy   DormancyPolicy
	Scanner    *Scanner // optional — derived from Registry when nil
	Sizer      *Sizer   // optional — default NumCPU workers
	Cleaner    *Cleaner // optional — requires Journal, Safety, Registry
	Journal    *journal.Journal
	Categories []detectors.Category // filter applied to detectors; nil = all enabled
	Workers    int                  // scanner/sizer worker count; 0 = NumCPU
}

// SetCategories rebuilds the internal scanner so it only emits matches for
// the given category set (an empty/nil slice means "every supported
// category"). It is the runtime hook the TUI uses after the user picks
// categories interactively. Workers is reused from the existing scanner.
func (e *Engine) SetCategories(cats []detectors.Category) {
	classifier := NewClassifier(e.Registry.Filter(cats))
	workers := 0
	if e.Scanner != nil {
		workers = e.Scanner.NumWorkers
	}
	e.Scanner = NewScanner(classifier, e.Safety, workers)
}

// EnabledCategories is the set of detectors currently exposed by the
// platform — the TUI uses this to render its picker.
func (e *Engine) EnabledCategories() []detectors.Detector {
	if e.Registry == nil {
		return nil
	}
	return e.Registry.Enabled()
}

// New builds an Engine from cfg, returning an error on invalid input.
func New(cfg Config) (*Engine, error) {
	if cfg.Registry == nil {
		return nil, errors.New("engine: Registry is required")
	}
	if cfg.Safety == nil {
		cfg.Safety = NewSafety()
	}
	classifier := NewClassifier(cfg.Registry.Filter(cfg.Categories))
	scanner := cfg.Scanner
	if scanner == nil {
		scanner = NewScanner(classifier, cfg.Safety, cfg.Workers)
	}
	sizer := cfg.Sizer
	if sizer == nil {
		sizer = NewSizer(cfg.Workers)
	}
	cleaner := cfg.Cleaner
	return &Engine{
		Registry: cfg.Registry,
		Safety:   cfg.Safety,
		Scanner:  scanner,
		Sizer:    sizer,
		Dormancy: cfg.Dormancy,
		Cleaner:  cleaner,
		Journal:  cfg.Journal,
	}, nil
}

// Scan walks the given roots and returns a channel of matches that honors the
// configured dormancy policy. The returned error channel emits at most one
// error. Both channels are closed when the scan completes.
func (e *Engine) Scan(ctx context.Context, roots []string) (<-chan detectors.Match, <-chan error) {
	matches := make(chan detectors.Match, 64)
	errc := make(chan error, 1)
	raw := make(chan detectors.Match, 64)
	go func() {
		errc <- e.Scanner.Scan(ctx, roots, raw)
		close(errc)
	}()
	go func() {
		defer close(matches)
		for m := range raw {
			if ctx.Err() != nil {
				return
			}
			if e.needsDormancy(m) && !e.Dormancy.IsDormant(ctx, m.Path, m.ModTime) {
				continue
			}
			select {
			case matches <- m:
			case <-ctx.Done():
				return
			}
		}
	}()
	return matches, errc
}

func (e *Engine) needsDormancy(m detectors.Match) bool {
	if e.Dormancy.MinAge <= 0 {
		return false
	}
	d := e.Registry.Get(m.Category)
	return d != nil && d.RequiresDormancy()
}

// SizeInPlace fills Match.SizeBytes for every item in matches, concurrently.
// It is a convenience wrapper around Sizer.SizeMany.
//
// Pseudo matches (e.g., docker:images) have no filesystem footprint and are
// skipped — their size remains zero and the cleanup summary will report
// bytes freed only after the native prune command runs.
func (e *Engine) SizeInPlace(ctx context.Context, matches []detectors.Match) {
	if len(matches) == 0 {
		return
	}
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		if m.IsPseudo() {
			continue
		}
		paths = append(paths, m.Path)
	}
	if len(paths) == 0 {
		return
	}
	sizes := make(map[string]int64, len(paths))
	var mu sync.Mutex
	e.Sizer.SizeMany(ctx, paths, func(path string, size int64, _ error) {
		mu.Lock()
		sizes[path] = size
		mu.Unlock()
	})
	for i := range matches {
		if matches[i].IsPseudo() {
			continue
		}
		matches[i].SizeBytes = sizes[matches[i].Path]
	}
}

// Clean runs the cleaner over a batch of matches and returns a CleanSummary.
// It never aborts on individual failures — per-match errors land in the
// result and in the journal.
func (e *Engine) Clean(ctx context.Context, matches []detectors.Match, opts detectors.CleanOptions) CleanSummary {
	sum := CleanSummary{
		StartedAt: time.Now(),
		DryRun:    opts.DryRun,
		Attempted: len(matches),
	}
	if e.Cleaner == nil {
		sum.Errors = append(sum.Errors, errors.New("engine: cleaner not configured"))
		sum.CompletedAt = time.Now()
		return sum
	}
	for _, m := range matches {
		if ctx.Err() != nil {
			break
		}
		res, _ := e.Cleaner.Clean(ctx, m, opts)
		if res.Err != nil {
			sum.Failed++
			sum.Errors = append(sum.Errors, res.Err)
			continue
		}
		sum.Succeeded++
		sum.BytesFreed += res.BytesFreed
	}
	sum.CompletedAt = time.Now()
	return sum
}

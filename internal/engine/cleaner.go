package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/journal"
	"github.com/guilhermejansen/clearstack/internal/trash"
)

// Cleaner executes cleanup strategies on matches while enforcing the safety
// whitelist and recording every attempt in the journal.
type Cleaner struct {
	Trash    trash.Trasher
	Journal  *journal.Journal
	Safety   *Safety
	Registry *detectors.Registry
}

// NewCleaner returns a Cleaner wired with the given dependencies. Any of
// them may be nil when not needed (e.g., journal disabled, no registry).
func NewCleaner(t trash.Trasher, j *journal.Journal, s *Safety, r *detectors.Registry) *Cleaner {
	return &Cleaner{Trash: t, Journal: j, Safety: s, Registry: r}
}

// Clean executes the strategy on m and writes the outcome to the journal.
// It returns the CleanResult (including an error field) and an error for
// unexpected failures that should abort a batch run.
func (c *Cleaner) Clean(ctx context.Context, m detectors.Match, opts detectors.CleanOptions) (detectors.CleanResult, error) {
	start := time.Now()
	result := detectors.CleanResult{
		Path:      m.Path,
		Category:  m.Category,
		Strategy:  m.Strategy,
		DryRun:    opts.DryRun,
		StartedAt: start,
	}
	strategy := m.Strategy
	if opts.Override != "" {
		strategy = opts.Override
		result.Strategy = strategy
	}

	// Safety gate — always, even for dry-run. Pseudo matches (e.g., Docker
	// prune targets like "docker:images") have no real filesystem path, so
	// safety validation is meaningless and would always fail; we skip it.
	if c.Safety != nil && !m.IsPseudo() {
		if err := c.Safety.Validate(m.Path); err != nil {
			result.Err = err
			result.CompletedAt = time.Now()
			c.record(result)
			return result, nil // non-fatal: a single protected path aborts this match only
		}
	}

	if opts.DryRun {
		result.BytesFreed = m.SizeBytes
		result.Undoable = true
		result.CompletedAt = time.Now()
		c.record(result)
		return result, nil
	}

	switch strategy {
	case detectors.StrategyTrash:
		if c.Trash == nil {
			result.Err = errors.New("cleaner: no trasher configured")
			break
		}
		r, err := c.Trash.Trash(m.Path)
		if err != nil {
			result.Err = fmt.Errorf("cleaner: trash: %w", err)
			break
		}
		result.BytesFreed = m.SizeBytes
		result.Undoable = r.Undoable
		result.UndoReference = r.TrashLocation
	case detectors.StrategyHardDelete:
		if err := os.RemoveAll(m.Path); err != nil {
			result.Err = fmt.Errorf("cleaner: remove: %w", err)
			break
		}
		result.BytesFreed = m.SizeBytes
	case detectors.StrategyNativeCommand:
		parsed, err := c.runNative(ctx, m)
		if err != nil {
			result.Err = err
			break
		}
		// Prefer detector-parsed bytes (e.g., docker prune's "Total
		// reclaimed space" trailer) over Match.SizeBytes — the latter is
		// always 0 for pseudo matches, so without this the summary would
		// always misleadingly report "0 B" for Docker prunes.
		if parsed > 0 {
			result.BytesFreed = parsed
		} else {
			result.BytesFreed = m.SizeBytes
		}
	case detectors.StrategyDockerAPI:
		result.Err = errors.New("cleaner: docker strategy requires DockerCleaner (Sprint 3)")
	case detectors.StrategyNoop:
		// Nothing to do.
	default:
		result.Err = fmt.Errorf("cleaner: unknown strategy %q", strategy)
	}

	result.CompletedAt = time.Now()
	c.record(result)
	return result, nil
}

// runNative executes a detector's native command and returns the bytes the
// detector parsed from its output (0 when no parser is implemented or the
// output didn't contain a parseable size). The caller decides whether to
// fall back to Match.SizeBytes for the BytesFreed report.
func (c *Cleaner) runNative(ctx context.Context, m detectors.Match) (int64, error) {
	if c.Registry == nil {
		return 0, errors.New("cleaner: registry is nil — cannot resolve native command")
	}
	d := c.Registry.Get(m.Category)
	if d == nil {
		return 0, fmt.Errorf("cleaner: detector %q not registered", m.Category)
	}
	nc, ok := d.(detectors.NativeCommander)
	if !ok {
		return 0, fmt.Errorf("cleaner: detector %q does not implement NativeCommander", m.Category)
	}
	argv, stdin := nc.NativeCommand(m)
	if len(argv) == 0 {
		return 0, fmt.Errorf("cleaner: detector %q returned empty argv", m.Category)
	}
	// #nosec G204 — argv comes exclusively from registered NativeCommander
	// implementations controlled by this binary. No user-supplied strings
	// flow into argv[0]; user input only influences the matched Path, which
	// detectors pass via env or --flag style args they fully control.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if m.ProjectRoot != "" {
		cmd.Dir = m.ProjectRoot
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("cleaner: %s failed: %w: %s", argv[0], err, strings.TrimSpace(string(out)))
	}
	if parser, ok := d.(detectors.NativeOutputParser); ok {
		return parser.ParseNativeOutput(string(out)), nil
	}
	return 0, nil
}

func (c *Cleaner) record(r detectors.CleanResult) {
	if c.Journal == nil {
		return
	}
	e := journal.Entry{
		Category:      string(r.Category),
		Strategy:      string(r.Strategy),
		OriginalPath:  r.Path,
		TrashLocation: r.UndoReference,
		BytesFreed:    r.BytesFreed,
		DryRun:        r.DryRun,
		Undoable:      r.Undoable,
	}
	if r.Err != nil {
		e.Err = r.Err.Error()
	}
	_ = c.Journal.Append(e)
}

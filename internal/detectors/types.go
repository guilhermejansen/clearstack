// Package detectors defines cleanup categories and the Detector interface used
// by the engine to identify and safely clean developer artifacts across
// multiple stacks and platforms.
package detectors

import (
	"context"
	"io/fs"
	"time"
)

// Category is a stable, machine-readable identifier for a cleanup category.
// Examples: "node_modules", "next_cache", "pycache", "go_cache", "docker_images".
type Category string

// Strategy describes how a matched artifact should be cleaned.
type Strategy string

const (
	// StrategyTrash moves the artifact to the OS recycle bin.
	StrategyTrash Strategy = "trash"
	// StrategyHardDelete removes the artifact with os.RemoveAll.
	StrategyHardDelete Strategy = "hard"
	// StrategyNativeCommand invokes an official tool
	// (e.g., "go clean -modcache", "pnpm store prune", "docker system prune").
	StrategyNativeCommand Strategy = "native"
	// StrategyDockerAPI uses the Docker Go client to prune resources.
	StrategyDockerAPI Strategy = "docker"
	// StrategyNoop performs no cleanup — used for read-only reporting detectors.
	StrategyNoop Strategy = "noop"
)

// Safety level assigned to a category. Higher levels require stronger consent.
type Safety int

const (
	// SafetySafe artifacts are always regenerable (e.g., node_modules, __pycache__).
	SafetySafe Safety = iota
	// SafetyCaution artifacts are regenerable but removal is slow or inconvenient
	// (e.g., Maven local repo, NuGet global cache).
	SafetyCaution
	// SafetyDanger artifacts can cause data loss or break other projects
	// if cleaned incorrectly (e.g., pnpm store via raw rm, Docker volumes).
	SafetyDanger
)

// String returns a lowercase label for Safety.
func (s Safety) String() string {
	switch s {
	case SafetySafe:
		return "safe"
	case SafetyCaution:
		return "caution"
	case SafetyDanger:
		return "danger"
	default:
		return "unknown"
	}
}

// Match represents a single artifact found on disk that belongs to a category.
type Match struct {
	// Path is the absolute, cleaned filesystem path of the matched artifact.
	Path string
	// Category is the detector category this match belongs to.
	Category Category
	// DetectorID points back to the Detector.ID() that produced this match.
	DetectorID string
	// Safety level copied from the detector at match time.
	Safety Safety
	// Strategy the detector will use to clean this match.
	Strategy Strategy
	// FoundAt records when the scanner emitted this match.
	FoundAt time.Time
	// ModTime is the mtime of the matched path, used for dormancy checks.
	ModTime time.Time
	// SizeBytes is the aggregate size; zero until the sizer computes it.
	SizeBytes int64
	// ProjectRoot is the containing project directory (e.g., repo root with
	// a lock file). Empty when not applicable.
	ProjectRoot string
}

// CleanOptions controls how a single match is cleaned.
type CleanOptions struct {
	// DryRun reports what would happen without modifying anything.
	DryRun bool
	// Override replaces the detector's default strategy when non-empty.
	Override Strategy
}

// CleanResult describes the outcome of cleaning a single match.
type CleanResult struct {
	Path          string
	Category      Category
	Strategy      Strategy
	BytesFreed    int64
	DryRun        bool
	Undoable      bool
	UndoReference string
	StartedAt     time.Time
	CompletedAt   time.Time
	Err           error
}

// Detector is the contract every cleanup category implements.
//
// A Detector is stateless and safe for concurrent use. Cleaning is handled
// centrally by the engine's Cleaner using the strategy declared on each Match;
// detectors that need a custom shell command implement the optional
// NativeCommander interface.
type Detector interface {
	// ID returns the stable category identifier.
	ID() Category
	// Description is a one-line human-readable summary.
	Description() string
	// Safety level of this category.
	Safety() Safety
	// DefaultStrategy is the strategy used when none is overridden.
	DefaultStrategy() Strategy
	// PlatformSupported reports whether this detector can run on the
	// current OS.
	PlatformSupported() bool
	// RequiresDormancy returns true when the engine should filter matches
	// by project dormancy (mtime older than the configured threshold).
	RequiresDormancy() bool
	// StopDescent returns true when the scanner should not descend into
	// a matched directory (e.g., node_modules — we never enter them).
	StopDescent() bool
	// Match is invoked by the scanner for every directory entry it sees.
	// It returns a non-nil Match when the entry belongs to this category.
	Match(ctx context.Context, path string, entry fs.DirEntry) *Match
}

// NativeCommander is an optional interface implemented by detectors whose
// cleanup uses an external tool (e.g., "go clean -modcache",
// "pnpm store prune", "docker system prune"). The Cleaner will call it when
// the Match.Strategy is StrategyNativeCommand.
type NativeCommander interface {
	// NativeCommand returns argv for exec.CommandContext and optional stdin.
	// The working directory (if relevant) is controlled by Match.ProjectRoot.
	NativeCommand(m Match) (argv []string, stdin string)
}

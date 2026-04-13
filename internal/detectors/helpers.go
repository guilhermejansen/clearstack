package detectors

import (
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
)

// SimpleDirDetector matches directories by exact name. It covers the common
// "stop-descent, safe trash" pattern shared by many categories
// (node_modules, .next, .turbo, __pycache__, target, ...).
type SimpleDirDetector struct {
	CategoryID       Category
	DirName          string
	Desc             string
	SafetyLevel      Safety
	Strategy         Strategy
	NeedsDormancy    bool
	Platforms        []string // empty = every OS
	StopDescentValue bool     // defaults to true; set to override
	// Require lets a detector gate a match on sibling files (e.g.,
	// node_modules only counts if a package.json sits next to it).
	Require []string
}

// ID returns the detector's category identifier.
func (s *SimpleDirDetector) ID() Category { return s.CategoryID }

// Description returns a one-line human-readable summary.
func (s *SimpleDirDetector) Description() string { return s.Desc }

// Safety returns the detector's safety level.
func (s *SimpleDirDetector) Safety() Safety { return s.SafetyLevel }

// DefaultStrategy returns the detector's cleanup strategy.
func (s *SimpleDirDetector) DefaultStrategy() Strategy { return s.Strategy }

// PlatformSupported reports whether the detector runs on the current OS.
func (s *SimpleDirDetector) PlatformSupported() bool {
	if len(s.Platforms) == 0 {
		return true
	}
	cur := runtime.GOOS
	for _, p := range s.Platforms {
		if p == cur {
			return true
		}
	}
	return false
}

// RequiresDormancy reports whether matches must be filtered by project idle time.
func (s *SimpleDirDetector) RequiresDormancy() bool { return s.NeedsDormancy }

// StopDescent reports whether the scanner should skip entering the matched dir.
func (s *SimpleDirDetector) StopDescent() bool {
	if !s.StopDescentValue {
		return true // sane default for container dirs
	}
	return s.StopDescentValue
}

// Match returns a non-nil Match when the entry is a directory with the
// configured name and all required sibling files exist.
func (s *SimpleDirDetector) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() || entry.Name() != s.DirName {
		return nil
	}
	parent := filepath.Dir(path)
	for _, req := range s.Require {
		if !pathExists(filepath.Join(parent, req)) {
			return nil
		}
	}
	return &Match{
		Path:     path,
		Category: s.CategoryID,
		Safety:   s.SafetyLevel,
		Strategy: s.Strategy,
	}
}

// pathExists is a package-local stat helper (separate from engine.statCached
// to avoid a circular import).
func pathExists(p string) bool {
	_, err := osLstat(p)
	return err == nil
}

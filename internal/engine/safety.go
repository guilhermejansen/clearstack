package engine

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/guilhermejansen/clearstack/internal/platform"
)

// ErrProtectedPath is returned when an operation targets a whitelisted path.
var ErrProtectedPath = errors.New("engine: path is protected by safety whitelist")

// Safety enforces a whitelist of paths that must never be modified or deleted.
//
// The whitelist covers obvious footguns (`/`, `/System`, the user's home
// itself, package manager roots) and is enforced at the Cleaner layer before
// any cleanup strategy runs.
type Safety struct {
	protected []string
}

// NewSafety builds a Safety with a sensible default whitelist for the current
// platform plus any extra paths provided by the caller (from config).
func NewSafety(extra ...string) *Safety {
	base := defaultProtected()
	for _, p := range extra {
		if p == "" {
			continue
		}
		base = append(base, filepath.Clean(platform.ExpandHome(p)))
	}
	return &Safety{protected: base}
}

// Protected returns a copy of the current whitelist for inspection.
func (s *Safety) Protected() []string {
	out := make([]string, len(s.protected))
	copy(out, s.protected)
	return out
}

// Validate returns nil when path is safe to clean, or ErrProtectedPath wrapping
// a descriptive error otherwise.
//
// A path is unsafe when it:
//   - is empty or relative,
//   - matches a protected path exactly, or
//   - IS a parent of a protected path (e.g., trying to delete ~/ while ~/Library
//     is protected).
//
// Children of protected paths that belong to known sub-areas (e.g.,
// ~/Library/Developer/Xcode/DerivedData) are handled by individual detectors
// which opt into specific subpaths explicitly, so Safety never needs to allow
// them generically.
func (s *Safety) Validate(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrProtectedPath)
	}
	abs := filepath.Clean(path)
	if !filepath.IsAbs(abs) {
		return fmt.Errorf("%w: path must be absolute: %q", ErrProtectedPath, path)
	}
	for _, p := range s.protected {
		if platform.PathEqual(abs, p) {
			return fmt.Errorf("%w: %q is whitelisted", ErrProtectedPath, abs)
		}
		if platform.PathHasPrefix(p, abs) && !platform.PathEqual(p, abs) {
			// abs is an ancestor of a protected path → refuse.
			return fmt.Errorf("%w: %q is an ancestor of protected %q", ErrProtectedPath, abs, p)
		}
	}
	return nil
}

// IsProtected is a convenience predicate equivalent to Validate(path) == nil.
func (s *Safety) IsProtected(path string) bool {
	return s.Validate(path) != nil
}

func defaultProtected() []string {
	home := platform.Home()
	var base []string
	base = append(base,
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/lib",
		"/opt",
		"/proc",
		"/root",
		"/sbin",
		"/srv",
		"/sys",
		"/tmp",
		"/usr",
		"/var",
	)
	if platform.IsDarwin() {
		base = append(base,
			"/Applications",
			"/Library",
			"/System",
			"/Users",
			"/private",
			"/Volumes",
		)
	}
	if platform.IsWindows() {
		base = append(base,
			`C:\`,
			`C:\Windows`,
			`C:\Program Files`,
			`C:\Program Files (x86)`,
			`C:\Users`,
			`C:\ProgramData`,
		)
	}
	if home != "" {
		base = append(base, home)
	}
	// Normalize.
	seen := make(map[string]struct{}, len(base))
	out := make([]string, 0, len(base))
	for _, p := range base {
		if p == "" {
			continue
		}
		c := filepath.Clean(p)
		key := c
		if platform.IsWindows() {
			key = strings.ToLower(c)
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}

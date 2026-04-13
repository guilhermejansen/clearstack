// Package trash moves filesystem entries to the OS recycle bin in a
// cross-platform and reversible way.
//
// Each platform is implemented in its own build-tag-guarded file:
//
//	trash_darwin.go   — Finder via osascript
//	trash_linux.go    — freedesktop.org Trash spec in $XDG_DATA_HOME/Trash
//	trash_windows.go  — SHFileOperationW with FOF_ALLOWUNDO
//
// A fallback implementation (trash_fallback.go) moves files to
// ~/.local/state/clearstack/trash/<timestamp>/ when no platform trash is
// available. This is picked up automatically by New() when the native
// implementation returns ErrUnsupported.
package trash

import (
	"errors"
	"time"
)

// ErrUnsupported is returned by a platform implementation that cannot trash
// the given path (e.g., different filesystem on Linux, missing Finder on
// headless macOS). The package transparently switches to the fallback.
var ErrUnsupported = errors.New("trash: not supported on this platform for this path")

// Receipt describes a single trash operation.
type Receipt struct {
	// OriginalPath is the absolute path of the file/dir before trashing.
	OriginalPath string
	// TrashLocation is the post-trash location when known (e.g., the
	// Linux trash info entry or the fallback archive directory). It may be
	// empty on platforms that do not expose it (macOS Finder).
	TrashLocation string
	// TrashedAt is when the operation completed.
	TrashedAt time.Time
	// Undoable reports whether the receipt can be used to restore the entry.
	Undoable bool
}

// Trasher is the portable interface implemented per-OS.
type Trasher interface {
	// Trash moves path to the OS recycle bin and returns a Receipt.
	Trash(path string) (Receipt, error)
	// Name returns the implementation label (for logs and doctor).
	Name() string
}

// New returns the platform-native Trasher, falling back to a filesystem
// archive when no native implementation is available.
func New() Trasher {
	if t := newNative(); t != nil {
		return t
	}
	return newFallback()
}

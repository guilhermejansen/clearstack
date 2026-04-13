package trash

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guilhermejansen/clearstack/internal/platform"
)

// fallbackTrasher moves entries into a clearstack-managed archive under the
// platform state directory. It is used when a native implementation is
// unavailable or when native trash returns ErrUnsupported (e.g., cross-device
// move on Linux).
type fallbackTrasher struct {
	root string
}

func newFallback() Trasher {
	return &fallbackTrasher{root: filepath.Join(platform.StateDir(), "trash")}
}

func (*fallbackTrasher) Name() string { return "fallback-archive" }

func (f *fallbackTrasher) Trash(path string) (Receipt, error) {
	if path == "" {
		return Receipt{}, fmt.Errorf("trash: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Receipt{}, fmt.Errorf("trash: absolute path: %w", err)
	}
	if _, err := os.Lstat(abs); err != nil {
		return Receipt{}, fmt.Errorf("trash: stat: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	mirror := strings.TrimPrefix(abs, string(filepath.Separator))
	mirror = strings.ReplaceAll(mirror, string(filepath.Separator), "__")
	target := filepath.Join(f.root, stamp, mirror)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return Receipt{}, fmt.Errorf("trash: mkdir: %w", err)
	}
	if err := os.Rename(abs, target); err != nil {
		return Receipt{}, fmt.Errorf("trash: rename: %w", err)
	}
	return Receipt{
		OriginalPath:  abs,
		TrashLocation: target,
		TrashedAt:     time.Now(),
		Undoable:      true,
	}, nil
}

//go:build linux

package trash

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/adrg/xdg"
)

type xdgTrasher struct {
	// counter disambiguates collisions within the same millisecond.
	counter int64
}

func newNative() Trasher { return &xdgTrasher{} }

func (*xdgTrasher) Name() string { return "xdg-trash" }

// Trash implements the freedesktop.org Trash specification v1.0.
// https://specifications.freedesktop.org/trash-spec/trashspec-latest.html
//
// For files on the user's home partition it uses $XDG_DATA_HOME/Trash
// (typically ~/.local/share/Trash). For files on other filesystems it
// returns ErrUnsupported so the caller falls back to the archive strategy;
// proper top-level $topdir/.Trash support is left for a later pass.
func (t *xdgTrasher) Trash(path string) (Receipt, error) {
	if path == "" {
		return Receipt{}, fmt.Errorf("trash: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Receipt{}, fmt.Errorf("trash: absolute path: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return Receipt{}, fmt.Errorf("trash: stat: %w", err)
	}
	trashRoot := filepath.Join(xdg.DataHome, "Trash")
	home, err := os.UserHomeDir()
	if err != nil || !strings.HasPrefix(abs, filepath.Clean(home)+string(filepath.Separator)) {
		return Receipt{}, ErrUnsupported
	}
	filesDir := filepath.Join(trashRoot, "files")
	infoDir := filepath.Join(trashRoot, "info")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return Receipt{}, fmt.Errorf("trash: mkdir files: %w", err)
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return Receipt{}, fmt.Errorf("trash: mkdir info: %w", err)
	}

	base := filepath.Base(abs)
	name := base
	n := atomic.AddInt64(&t.counter, 1)
	for i := 0; i < 64; i++ {
		candidate := filepath.Join(filesDir, name)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			break
		}
		name = fmt.Sprintf("%s.%d.%d", base, time.Now().UnixNano(), n+int64(i))
	}

	targetFile := filepath.Join(filesDir, name)
	targetInfo := filepath.Join(infoDir, name+".trashinfo")

	if err := os.Rename(abs, targetFile); err != nil {
		return Receipt{}, ErrUnsupported // cross-device or other — fall back
	}

	deletionDate := info.ModTime().Format("2006-01-02T15:04:05")
	content := fmt.Sprintf(
		"[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		url.PathEscape(abs),
		deletionDate,
	)
	if err := os.WriteFile(targetInfo, []byte(content), 0o600); err != nil {
		// Leave the moved file in place; the info sidecar is advisory.
		return Receipt{OriginalPath: abs, TrashLocation: targetFile, TrashedAt: time.Now(), Undoable: true}, nil
	}
	return Receipt{
		OriginalPath:  abs,
		TrashLocation: targetFile,
		TrashedAt:     time.Now(),
		Undoable:      true,
	}, nil
}

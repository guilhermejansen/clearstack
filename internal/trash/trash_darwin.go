//go:build darwin

package trash

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type finderTrasher struct{}

func newNative() Trasher { return &finderTrasher{} }

func (finderTrasher) Name() string { return "macos-finder" }

func (finderTrasher) Trash(path string) (Receipt, error) {
	if path == "" {
		return Receipt{}, fmt.Errorf("trash: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Receipt{}, fmt.Errorf("trash: absolute path: %w", err)
	}
	// Use Finder AppleScript — moves to the user's Trash in Finder, fully
	// reversible by the user via "Put Back".
	script := fmt.Sprintf(
		`tell application "Finder" to move POSIX file %q to trash`,
		abs,
	)
	cmd := exec.Command("osascript", "-e", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return Receipt{}, fmt.Errorf("trash: osascript failed: %s", msg)
	}
	return Receipt{
		OriginalPath: abs,
		TrashedAt:    time.Now(),
		Undoable:     true,
	}, nil
}

//go:build windows

package trash

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

// windowsTrasher uses Windows PowerShell to move paths to the Recycle Bin via
// Microsoft.VisualBasic.FileIO.FileSystem.DeleteFile / DeleteDirectory, which
// honors the Windows Recycle Bin (fully reversible by the user).
//
// A native SHFileOperationW-based implementation is feasible and avoids the
// PowerShell dependency, but PowerShell ships with every supported Windows
// SKU and avoids cgo, so we use it here. Sprint 5+ may swap in syscall-based
// code when hardening for non-interactive environments.
type windowsTrasher struct{}

func newNative() Trasher { return &windowsTrasher{} }

func (windowsTrasher) Name() string { return "windows-recycle-bin" }

func (windowsTrasher) Trash(path string) (Receipt, error) {
	if path == "" {
		return Receipt{}, fmt.Errorf("trash: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Receipt{}, fmt.Errorf("trash: absolute path: %w", err)
	}
	// DeleteDirectory handles folders; DeleteFile handles files. We try both
	// with runtime discrimination via a short Test-Path probe to avoid
	// PowerShell errors that print to stderr.
	script := fmt.Sprintf(`
Add-Type -AssemblyName Microsoft.VisualBasic
$p = %q
if (Test-Path -LiteralPath $p -PathType Container) {
  [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory(
    $p, 'OnlyErrorDialogs', 'SendToRecycleBin')
} else {
  [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile(
    $p, 'OnlyErrorDialogs', 'SendToRecycleBin')
}`, abs)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Receipt{}, fmt.Errorf("trash: powershell: %v: %s", err, stderr.String())
	}
	return Receipt{
		OriginalPath: abs,
		TrashedAt:    time.Now(),
		Undoable:     true,
	}, nil
}

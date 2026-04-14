package detectors

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// pnpm store cleanup is special: pnpm symlinks from every project back into
// the global store, so a naive `rm -rf` breaks every installed project on
// the machine. The only safe way is `pnpm store prune`, which the binary
// uses exclusively.

func init() {
	Register(&pnpmStore{})
}

type pnpmStore struct {
	once sync.Once
	path string
	ok   bool
}

func (*pnpmStore) ID() Category { return "pnpm_store" }
func (*pnpmStore) Description() string {
	return "pnpm content-addressable store (cleaned via `pnpm store prune`)"
}
func (*pnpmStore) Safety() Safety            { return SafetyCaution }
func (*pnpmStore) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (*pnpmStore) RequiresDormancy() bool    { return false }
func (*pnpmStore) StopDescent() bool         { return true }

func (*pnpmStore) PlatformSupported() bool {
	_, err := exec.LookPath("pnpm")
	return err == nil
}

func (p *pnpmStore) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	p.once.Do(p.resolve)
	if !p.ok {
		return nil
	}
	if path != p.path {
		return nil
	}
	return &Match{
		Path:     path,
		Category: p.ID(),
		Safety:   SafetyCaution,
		Strategy: StrategyNativeCommand,
	}
}

// NativeCommand always invokes `pnpm store prune`. The Match.Path is just
// informational — pnpm figures out the store itself.
func (*pnpmStore) NativeCommand(_ Match) ([]string, string) {
	return []string{"pnpm", "store", "prune"}, ""
}

func (p *pnpmStore) resolve() {
	// Typical defaults; pnpm respects STORE_DIR if set but we only need a
	// canonical location to match against during scans.
	var candidates []string
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			candidates = append(candidates, filepath.Join(v, "pnpm", "store"))
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "pnpm", "store"),
			filepath.Join(home, "Library", "pnpm", "store"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Lstat(c); err == nil {
			p.path = filepath.Clean(strings.TrimSpace(c))
			p.ok = true
			return
		}
	}
}

package detectors

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// bun install cache is regenerable and safe to trash directly; unlike pnpm,
// bun uses copy-on-write (or hardlinks) so removing the cache directory
// does not break any already-installed project.

func init() {
	Register(&bunCache{})
}

type bunCache struct {
	once sync.Once
	path string
	ok   bool
}

func (*bunCache) ID() Category              { return "bun_install_cache" }
func (*bunCache) Description() string       { return "bun install cache (~/.bun/install/cache)" }
func (*bunCache) Safety() Safety            { return SafetySafe }
func (*bunCache) DefaultStrategy() Strategy { return StrategyTrash }
func (*bunCache) RequiresDormancy() bool    { return false }
func (*bunCache) StopDescent() bool         { return true }
func (*bunCache) PlatformSupported() bool {
	_, err := exec.LookPath("bun")
	return err == nil
}

func (b *bunCache) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	b.once.Do(b.resolve)
	if !b.ok || path != b.path {
		return nil
	}
	return &Match{Path: path, Category: b.ID(), Safety: SafetySafe, Strategy: StrategyTrash}
}

func (b *bunCache) resolve() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidate := filepath.Join(home, ".bun", "install", "cache")
	if _, err := os.Lstat(candidate); err == nil {
		b.path = filepath.Clean(candidate)
		b.ok = true
	}
}

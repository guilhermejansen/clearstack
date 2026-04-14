package detectors

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// pip, poetry and uv all maintain separate global download caches. We wrap
// each tool's official `cache clean` (or `cache purge`) command so download
// integrity is preserved.

func init() {
	Register(&pipCache{})
	Register(&poetryCache{})
	Register(&uvCache{})
}

type pipCache struct {
	once sync.Once
	path string
	ok   bool
}

func (*pipCache) ID() Category              { return "pip_cache" }
func (*pipCache) Description() string       { return "pip download cache (cleaned via `pip cache purge`)" }
func (*pipCache) Safety() Safety            { return SafetySafe }
func (*pipCache) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (*pipCache) RequiresDormancy() bool    { return false }
func (*pipCache) StopDescent() bool         { return true }
func (*pipCache) PlatformSupported() bool {
	_, err := exec.LookPath("pip")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("pip3")
	return err == nil
}

func (p *pipCache) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	p.once.Do(p.resolve)
	if !p.ok || path != p.path {
		return nil
	}
	return &Match{Path: path, Category: p.ID(), Safety: SafetySafe, Strategy: StrategyNativeCommand}
}

func (*pipCache) NativeCommand(_ Match) ([]string, string) {
	// pip cache purge is idempotent across pip/pip3.
	if _, err := exec.LookPath("pip"); err == nil {
		return []string{"pip", "cache", "purge"}, ""
	}
	return []string{"pip3", "cache", "purge"}, ""
}

func (p *pipCache) resolve() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates, filepath.Join(home, "Library", "Caches", "pip"))
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			candidates = append(candidates, filepath.Join(v, "pip", "Cache"))
		}
	default:
		candidates = append(candidates, filepath.Join(home, ".cache", "pip"))
	}
	for _, c := range candidates {
		if _, err := os.Lstat(c); err == nil {
			p.path = filepath.Clean(c)
			p.ok = true
			return
		}
	}
}

type poetryCache struct {
	once sync.Once
	path string
	ok   bool
}

func (*poetryCache) ID() Category { return "poetry_cache" }
func (*poetryCache) Description() string {
	return "Poetry cache (cleaned via `poetry cache clear --all .`)"
}
func (*poetryCache) Safety() Safety            { return SafetySafe }
func (*poetryCache) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (*poetryCache) RequiresDormancy() bool    { return false }
func (*poetryCache) StopDescent() bool         { return true }
func (*poetryCache) PlatformSupported() bool {
	_, err := exec.LookPath("poetry")
	return err == nil
}

func (p *poetryCache) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	p.once.Do(p.resolve)
	if !p.ok || path != p.path {
		return nil
	}
	return &Match{Path: path, Category: p.ID(), Safety: SafetySafe, Strategy: StrategyNativeCommand}
}

func (*poetryCache) NativeCommand(_ Match) ([]string, string) {
	// Poetry requires the dot to indicate "all caches" — see poetry#8156.
	return []string{"poetry", "cache", "clear", "--all", "."}, "y\n"
}

func (p *poetryCache) resolve() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidate := filepath.Join(home, ".cache", "pypoetry")
	if runtime.GOOS == "darwin" {
		candidate = filepath.Join(home, "Library", "Caches", "pypoetry")
	}
	if _, err := os.Lstat(candidate); err == nil {
		p.path = filepath.Clean(candidate)
		p.ok = true
	}
}

type uvCache struct {
	once sync.Once
	path string
	ok   bool
}

func (*uvCache) ID() Category              { return "uv_cache" }
func (*uvCache) Description() string       { return "uv cache (cleaned via `uv cache clean`)" }
func (*uvCache) Safety() Safety            { return SafetySafe }
func (*uvCache) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (*uvCache) RequiresDormancy() bool    { return false }
func (*uvCache) StopDescent() bool         { return true }
func (*uvCache) PlatformSupported() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}

func (u *uvCache) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	u.once.Do(u.resolve)
	if !u.ok || path != u.path {
		return nil
	}
	return &Match{Path: path, Category: u.ID(), Safety: SafetySafe, Strategy: StrategyNativeCommand}
}

func (*uvCache) NativeCommand(_ Match) ([]string, string) {
	return []string{"uv", "cache", "clean"}, ""
}

func (u *uvCache) resolve() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidate := filepath.Join(home, ".cache", "uv")
	if runtime.GOOS == "darwin" {
		candidate = filepath.Join(home, "Library", "Caches", "uv")
	}
	if _, err := os.Lstat(candidate); err == nil {
		u.path = filepath.Clean(candidate)
		u.ok = true
	}
}

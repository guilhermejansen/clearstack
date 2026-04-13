package detectors

import (
	"context"
	"io/fs"
	"os/exec"
	"strings"
	"sync"
)

// Go tooling caches (GOCACHE, GOMODCACHE, test cache) are cleaned via the
// official `go clean` subcommands — never with raw rm -rf.
//
// `go clean -modcache` is mandatory: files under GOMODCACHE are written
// read-only by the toolchain and naive deletion fails in non-obvious ways.

func init() {
	Register(&goCleanCache{category: "go_build_cache", flag: "-cache", desc: "Go build cache (GOCACHE)"})
	Register(&goCleanCache{category: "go_mod_cache", flag: "-modcache", desc: "Go module download cache (GOMODCACHE)"})
	Register(&goCleanCache{category: "go_test_cache", flag: "-testcache", desc: "Go test result cache"})
}

// goCleanCache is a singleton detector: it emits exactly one Match (the
// resolved cache path) when scanned, and cleans it by invoking
// `go clean <flag>`.
type goCleanCache struct {
	category Category
	flag     string // "-cache", "-modcache", "-testcache"
	desc     string

	once sync.Once
	path string
	ok   bool
}

func (g *goCleanCache) ID() Category              { return g.category }
func (g *goCleanCache) Description() string       { return g.desc }
func (g *goCleanCache) Safety() Safety            { return SafetySafe }
func (g *goCleanCache) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (g *goCleanCache) RequiresDormancy() bool    { return false }
func (g *goCleanCache) StopDescent() bool         { return true }

func (g *goCleanCache) PlatformSupported() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// Match returns a Match when the current directory equals the Go cache root
// this detector targets. It only fires once per distinct root per scan.
func (g *goCleanCache) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	g.once.Do(g.resolve)
	if !g.ok || g.path == "" {
		return nil
	}
	if path != g.path {
		return nil
	}
	return &Match{
		Path:     path,
		Category: g.category,
		Safety:   SafetySafe,
		Strategy: StrategyNativeCommand,
	}
}

// NativeCommand returns the `go clean -<flag>` argv.
func (g *goCleanCache) NativeCommand(_ Match) ([]string, string) {
	return []string{"go", "clean", g.flag}, ""
}

func (g *goCleanCache) resolve() {
	var env string
	switch g.flag {
	case "-cache":
		env = "GOCACHE"
	case "-modcache":
		env = "GOMODCACHE"
	case "-testcache":
		// no direct env var — skip the scanner, caller can still force-clean
		// via `clearstack clean --categories=go_test_cache`.
		return
	default:
		return
	}
	out, err := exec.Command("go", "env", env).Output()
	if err != nil {
		return
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return
	}
	g.path = p
	g.ok = true
}

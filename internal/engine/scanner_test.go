package engine

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

// buildFixture creates a synthetic project tree and returns the root.
//
//	root/
//	├── appA/
//	│   ├── package.json
//	│   └── node_modules/ (should match)
//	│       └── .bin/  (should NOT be visited)
//	├── appB/
//	│   ├── Cargo.toml
//	│   └── target/ (should match when rust detector is enabled)
//	└── appC/
//	    └── __pycache__/ (should match)
func buildFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mustMkdir(t, filepath.Join(root, "appA"))
	mustWrite(t, filepath.Join(root, "appA", "package.json"), "{}")
	mustMkdir(t, filepath.Join(root, "appA", "node_modules"))
	mustMkdir(t, filepath.Join(root, "appA", "node_modules", ".bin"))
	mustWrite(t, filepath.Join(root, "appA", "node_modules", "index.js"), "/* x */")
	mustWrite(t, filepath.Join(root, "appA", "node_modules", ".bin", "tool"), "#!/bin/sh")

	mustMkdir(t, filepath.Join(root, "appB"))
	mustWrite(t, filepath.Join(root, "appB", "Cargo.toml"), "[package]\nname=\"x\"\n")
	mustMkdir(t, filepath.Join(root, "appB", "target"))
	mustWrite(t, filepath.Join(root, "appB", "target", "binary"), "x")

	mustMkdir(t, filepath.Join(root, "appC"))
	mustMkdir(t, filepath.Join(root, "appC", "__pycache__"))
	mustWrite(t, filepath.Join(root, "appC", "__pycache__", "m.cpython-312.pyc"), "")

	return root
}

func TestScanner_BasicMatching(t *testing.T) {
	root := buildFixture(t)

	// Pick specific detectors to avoid filesystem-wide singletons
	// (Go caches, Xcode DerivedData) polluting the test set.
	ds := []detectors.Detector{
		detectors.Default.Get("node_modules"),
		detectors.Default.Get("rust_target"),
		detectors.Default.Get("pycache"),
	}
	for i, d := range ds {
		if d == nil {
			t.Fatalf("detector at index %d is nil — registry not populated", i)
		}
	}
	classifier := NewClassifier(ds)
	safety := NewSafety(root) // root must not be whitelisted for this test
	scanner := NewScanner(classifier, NewSafety(), 2)
	_ = safety

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan detectors.Match, 32)
	errCh := make(chan error, 1)
	go func() { errCh <- scanner.Scan(ctx, []string{root}, out) }()

	var got []detectors.Match
	for m := range out {
		got = append(got, m)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("scan error: %v", err)
	}

	sort.Slice(got, func(i, j int) bool { return got[i].Path < got[j].Path })
	if len(got) != 3 {
		t.Fatalf("expected 3 matches, got %d: %+v", len(got), got)
	}
	categories := map[detectors.Category]bool{}
	for _, m := range got {
		categories[m.Category] = true
	}
	wantCats := []detectors.Category{"node_modules", "pycache", "rust_target"}
	for _, c := range wantCats {
		if !categories[c] {
			t.Errorf("missing category %q in scan results", c)
		}
	}

	// StopDescent check: node_modules/.bin must NOT appear in results.
	for _, m := range got {
		if filepath.Base(m.Path) == ".bin" {
			t.Errorf("scanner should have stopped descending into node_modules, but visited %s", m.Path)
		}
	}
}

func TestScanner_RespectsContextCancellation(t *testing.T) {
	root := buildFixture(t)

	classifier := NewClassifier(detectors.Default.All())
	scanner := NewScanner(classifier, NewSafety(), 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before starting

	out := make(chan detectors.Match, 1)
	errCh := make(chan error, 1)
	go func() { errCh <- scanner.Scan(ctx, []string{root}, out) }()

	// Drain.
	for range out {
	}
	<-errCh // err may or may not be context.Canceled; the important thing is termination
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

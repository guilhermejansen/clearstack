package detectors

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

type fakeEntry struct {
	name  string
	isDir bool
}

func (f fakeEntry) Name() string               { return f.name }
func (f fakeEntry) IsDir() bool                { return f.isDir }
func (f fakeEntry) Type() fs.FileMode          { return 0 }
func (f fakeEntry) Info() (fs.FileInfo, error) { return nil, os.ErrInvalid }

func TestSimpleDirDetector_NameOnly(t *testing.T) {
	d := &SimpleDirDetector{
		CategoryID:  "node_modules",
		DirName:     "node_modules",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyTrash,
	}
	ctx := context.Background()

	if d.Match(ctx, "/tmp/x/node_modules", fakeEntry{name: "node_modules", isDir: true}) == nil {
		t.Error("expected match on directory named node_modules")
	}
	if d.Match(ctx, "/tmp/x/src", fakeEntry{name: "src", isDir: true}) != nil {
		t.Error("did not expect match on unrelated dir")
	}
	if d.Match(ctx, "/tmp/x/node_modules", fakeEntry{name: "node_modules", isDir: false}) != nil {
		t.Error("did not expect match on non-dir entry")
	}
}

func TestSimpleDirDetector_RequireFile(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(project, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Case A: package.json present → match.
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &SimpleDirDetector{
		CategoryID:  "node_modules",
		DirName:     "node_modules",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyTrash,
		Require:     []string{"package.json"},
	}
	if d.Match(context.Background(), filepath.Join(project, "node_modules"),
		fakeEntry{name: "node_modules", isDir: true}) == nil {
		t.Error("expected match when package.json is present")
	}

	// Case B: package.json removed → no match.
	_ = os.Remove(filepath.Join(project, "package.json"))
	if d.Match(context.Background(), filepath.Join(project, "node_modules"),
		fakeEntry{name: "node_modules", isDir: true}) != nil {
		t.Error("no match expected when required sibling file is missing")
	}
}

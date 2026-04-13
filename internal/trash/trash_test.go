package trash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackTrasher_MovesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_STATE_HOME", filepath.Join(dir, "state"))
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "local"))

	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("bye"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	tr := newFallback()
	r, err := tr.Trash(victim)
	if err != nil {
		t.Fatalf("Trash: %v", err)
	}
	if r.OriginalPath != victim {
		t.Errorf("OriginalPath = %q, want %q", r.OriginalPath, victim)
	}
	if r.TrashLocation == "" {
		t.Fatal("TrashLocation should be set by fallback")
	}
	if !r.Undoable {
		t.Error("fallback should be undoable")
	}
	if _, err := os.Lstat(victim); !os.IsNotExist(err) {
		t.Errorf("victim should be moved, but still exists: %v", err)
	}
	if _, err := os.Lstat(r.TrashLocation); err != nil {
		t.Errorf("trashed file should exist at %q: %v", r.TrashLocation, err)
	}
}

func TestFallbackTrasher_MissingFile(t *testing.T) {
	tr := newFallback()
	if _, err := tr.Trash("/definitely/does/not/exist-xyz"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

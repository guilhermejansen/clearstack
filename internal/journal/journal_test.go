package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJournal_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	// macOS ignores XDG; point the state explicitly.
	t.Setenv("HOME", dir)

	j, err := Open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = j.Close() }()

	e := Entry{
		Category:     "node_modules",
		Strategy:     "trash",
		OriginalPath: "/tmp/foo",
		BytesFreed:   42,
		Undoable:     true,
	}
	if err := j.Append(e); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Peek the raw file to confirm valid JSONL.
	data, err := os.ReadFile(j.Path())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Category != "node_modules" {
		t.Errorf("got category=%q, want node_modules", got.Category)
	}
	if got.ID == "" {
		t.Error("ID should have been auto-generated")
	}
	if got.At.IsZero() {
		t.Error("At should have been auto-populated")
	}

	// Read() helper should return the same entry.
	entries, err := Read(j.Path())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].BytesFreed != 42 {
		t.Errorf("BytesFreed = %d, want 42", entries[0].BytesFreed)
	}
	// Silence the unused import on some platforms.
	_ = filepath.Separator
}

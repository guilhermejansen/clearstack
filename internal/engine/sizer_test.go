package engine

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSizer_Size(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), "hello") // 5 bytes
	mustMkdir(t, filepath.Join(root, "sub"))
	mustWrite(t, filepath.Join(root, "sub", "b"), "world!") // 6 bytes

	s := NewSizer(2)
	got, err := s.Size(context.Background(), root)
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	const want = int64(11)
	if got != want {
		t.Errorf("size = %d, want %d", got, want)
	}
}

func TestSizer_EmptyPath(t *testing.T) {
	if _, err := NewSizer(1).Size(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

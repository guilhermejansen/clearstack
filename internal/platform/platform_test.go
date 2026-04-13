package platform

import (
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	h := Home()
	if h == "" {
		t.Skip("no home directory detected")
	}
	cases := []struct {
		in, want string
	}{
		{"~", h},
		{"~/foo", filepath.Join(h, "foo")},
		{"/abs", "/abs"},
		{"", ""},
	}
	for _, c := range cases {
		got := ExpandHome(c.in)
		if got != c.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathHasPrefix(t *testing.T) {
	if !PathHasPrefix("/a/b/c", "/a/b") {
		t.Errorf("/a/b/c should be under /a/b")
	}
	if PathHasPrefix("/ab", "/a") {
		t.Errorf("/ab should not be under /a")
	}
	if !PathHasPrefix("/a/b", "/a/b") {
		t.Errorf("equal paths should satisfy PathHasPrefix")
	}
}

func TestStateConfigCacheDirNonEmpty(t *testing.T) {
	if StateDir() == "" {
		t.Errorf("StateDir should not be empty")
	}
	if ConfigDir() == "" {
		t.Errorf("ConfigDir should not be empty")
	}
	if CacheDir() == "" {
		t.Errorf("CacheDir should not be empty")
	}
}

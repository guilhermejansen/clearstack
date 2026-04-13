package engine

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSafety_Validate_Whitelist(t *testing.T) {
	s := NewSafety()

	protected := "/"
	if runtime.GOOS == "darwin" {
		protected = "/System"
	}
	if err := s.Validate(protected); err == nil {
		t.Fatalf("expected %q to be rejected by default whitelist", protected)
	} else if !errors.Is(err, ErrProtectedPath) {
		t.Fatalf("expected ErrProtectedPath, got %v", err)
	}
}

func TestSafety_Validate_RejectsRelative(t *testing.T) {
	s := NewSafety()
	if err := s.Validate("relative/path"); err == nil {
		t.Fatal("expected relative path to be rejected")
	}
}

func TestSafety_Validate_RejectsEmpty(t *testing.T) {
	s := NewSafety()
	if err := s.Validate(""); err == nil {
		t.Fatal("expected empty path to be rejected")
	}
}

func TestSafety_Validate_AllowsSubpath(t *testing.T) {
	s := NewSafety()
	dir := t.TempDir()
	inside := filepath.Join(dir, "node_modules")
	if err := s.Validate(inside); err != nil {
		t.Fatalf("expected %q to be allowed, got %v", inside, err)
	}
}

func TestSafety_Validate_RefusesAncestorOfProtected(t *testing.T) {
	// A user-supplied ancestor of a protected path must be refused.
	// On Linux, /usr is protected; trying to clean / (its ancestor) must fail.
	s := NewSafety()
	if err := s.Validate("/"); err == nil {
		t.Fatal("expected / to be rejected as ancestor of protected paths")
	}
}

func TestSafety_Validate_ExtraWhitelist(t *testing.T) {
	extra := t.TempDir()
	s := NewSafety(extra)
	if err := s.Validate(extra); err == nil {
		t.Fatalf("expected custom whitelist %q to be enforced", extra)
	}
}

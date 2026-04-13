package detectors

import (
	"context"
	"io/fs"
	"testing"
)

type fakeDetector struct {
	id       Category
	platform bool
}

func (f fakeDetector) ID() Category              { return f.id }
func (f fakeDetector) Description() string       { return "fake" }
func (f fakeDetector) Safety() Safety            { return SafetySafe }
func (f fakeDetector) DefaultStrategy() Strategy { return StrategyTrash }
func (f fakeDetector) PlatformSupported() bool   { return f.platform }
func (f fakeDetector) RequiresDormancy() bool    { return false }
func (f fakeDetector) StopDescent() bool         { return true }
func (f fakeDetector) Match(_ context.Context, _ string, _ fs.DirEntry) *Match {
	return nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeDetector{id: "a", platform: true}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if d := r.Get("a"); d == nil {
		t.Fatal("Get returned nil for registered id")
	}
	if r.Get("missing") != nil {
		t.Fatal("Get should return nil for unknown id")
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(fakeDetector{id: "a", platform: true})
	if err := r.Register(fakeDetector{id: "a", platform: true}); err == nil {
		t.Fatal("expected duplicate registration to error")
	}
}

func TestRegistry_EnabledFiltersPlatform(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(fakeDetector{id: "on", platform: true})
	_ = r.Register(fakeDetector{id: "off", platform: false})
	en := r.Enabled()
	if len(en) != 1 || en[0].ID() != "on" {
		t.Fatalf("Enabled = %v, want only 'on'", en)
	}
}

func TestRegistry_FilterEmpty(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(fakeDetector{id: "on", platform: true})
	if len(r.Filter(nil)) != 1 {
		t.Fatal("Filter(nil) should return all enabled")
	}
	if len(r.Filter([]Category{})) != 1 {
		t.Fatal("Filter([]) should return all enabled")
	}
}

func TestRegistry_FilterByID(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(fakeDetector{id: "a", platform: true})
	_ = r.Register(fakeDetector{id: "b", platform: true})
	got := r.Filter([]Category{"a"})
	if len(got) != 1 || got[0].ID() != "a" {
		t.Fatalf("Filter = %v, want only 'a'", got)
	}
}

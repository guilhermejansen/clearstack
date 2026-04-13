package detectors

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds the set of active Detector implementations.
//
// It is safe for concurrent reads after construction. Mutations are
// serialized through an internal mutex and should generally only happen
// at process boot time via package-level init() hooks.
type Registry struct {
	mu   sync.RWMutex
	byID map[Category]Detector
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[Category]Detector)}
}

// Register adds a Detector to the registry. The second registration of the
// same Category replaces the first and returns an error describing the
// collision — callers should treat this as a bug.
func (r *Registry) Register(d Detector) error {
	if d == nil {
		return fmt.Errorf("detectors: cannot register nil detector")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := d.ID()
	if id == "" {
		return fmt.Errorf("detectors: detector has empty ID")
	}
	if _, dup := r.byID[id]; dup {
		r.byID[id] = d
		return fmt.Errorf("detectors: duplicate registration for category %q", id)
	}
	r.byID[id] = d
	return nil
}

// MustRegister panics if registration fails.
func (r *Registry) MustRegister(d Detector) {
	if err := r.Register(d); err != nil {
		panic(err)
	}
}

// Get returns the Detector for the given category, or nil if absent.
func (r *Registry) Get(id Category) Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[id]
}

// Has reports whether a detector is registered for id.
func (r *Registry) Has(id Category) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byID[id]
	return ok
}

// All returns every registered detector, sorted by category id for stable
// iteration (useful for reporting and tests).
func (r *Registry) All() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Detector, 0, len(r.byID))
	for _, d := range r.byID {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Enabled returns every registered detector that is supported on the current
// platform. The order matches All().
func (r *Registry) Enabled() []Detector {
	all := r.All()
	out := make([]Detector, 0, len(all))
	for _, d := range all {
		if d.PlatformSupported() {
			out = append(out, d)
		}
	}
	return out
}

// Filter returns only the detectors whose IDs are in the given set.
// Unknown IDs are silently ignored. Pass nil or empty to get everything.
func (r *Registry) Filter(ids []Category) []Detector {
	if len(ids) == 0 {
		return r.Enabled()
	}
	want := make(map[Category]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	all := r.All()
	out := make([]Detector, 0, len(all))
	for _, d := range all {
		if _, ok := want[d.ID()]; ok && d.PlatformSupported() {
			out = append(out, d)
		}
	}
	return out
}

// Default is the process-wide registry that detectors register into via init().
var Default = NewRegistry()

// Register is a convenience wrapper around Default.MustRegister used from
// package init() functions in detector implementations.
func Register(d Detector) { Default.MustRegister(d) }

package engine

import (
	"testing"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

func TestEngine_SetCategories_FiltersScanner(t *testing.T) {
	eng, err := New(Config{Registry: detectors.Default})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if eng.Scanner == nil || eng.Scanner.Classifier == nil {
		t.Fatal("expected scanner with classifier")
	}

	// Filter to a single category — only that detector should remain in
	// the scanner's classifier.
	eng.SetCategories([]detectors.Category{"node_modules"})
	if got := len(eng.Scanner.Classifier.Detectors()); got != 1 {
		t.Errorf("classifier detector count = %d after filter, want 1", got)
	}

	// Empty filter restores every supported detector.
	eng.SetCategories(nil)
	if got := len(eng.Scanner.Classifier.Detectors()); got <= 1 {
		t.Errorf("classifier detector count = %d after reset, want > 1", got)
	}
}

func TestEngine_SetCategories_PreservesWorkers(t *testing.T) {
	eng, err := New(Config{Registry: detectors.Default, Workers: 7})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if eng.Scanner.NumWorkers != 7 {
		t.Fatalf("initial workers = %d, want 7", eng.Scanner.NumWorkers)
	}
	eng.SetCategories([]detectors.Category{"next_cache"})
	if eng.Scanner.NumWorkers != 7 {
		t.Errorf("workers after SetCategories = %d, want 7 (preserved)", eng.Scanner.NumWorkers)
	}
}

func TestEngine_EnabledCategories_IncludesNewCategories(t *testing.T) {
	eng, err := New(Config{Registry: detectors.Default})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	cats := eng.EnabledCategories()
	if len(cats) == 0 {
		t.Fatal("EnabledCategories() returned nothing — Default registry empty?")
	}
	wantPresent := map[detectors.Category]bool{
		"terraform_dir":      true,
		"terragrunt_cache":   true,
		"docker_build_cache": true,
		"node_modules":       true,
	}
	seen := make(map[detectors.Category]bool, len(cats))
	for _, d := range cats {
		seen[d.ID()] = true
	}
	for id := range wantPresent {
		if !seen[id] {
			t.Errorf("expected %s in enabled categories", id)
		}
	}
}

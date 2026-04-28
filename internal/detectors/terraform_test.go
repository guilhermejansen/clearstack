package detectors

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTerraformDirDetector_RequiresTerraformSource(t *testing.T) {
	root := t.TempDir()
	stack := filepath.Join(root, "infra")
	if err := os.MkdirAll(filepath.Join(stack, ".terraform"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := terraformDirDetector{}
	ctx := context.Background()

	// No .tf sibling → no match.
	if got := d.Match(ctx, filepath.Join(stack, ".terraform"),
		fakeEntry{name: ".terraform", isDir: true}); got != nil {
		t.Errorf("no match expected when no *.tf sibling exists, got %+v", got)
	}

	// Add main.tf → match.
	if err := os.WriteFile(filepath.Join(stack, "main.tf"), []byte("terraform {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := d.Match(ctx, filepath.Join(stack, ".terraform"),
		fakeEntry{name: ".terraform", isDir: true})
	if got == nil {
		t.Fatal("expected match when *.tf sibling exists")
	}
	if got.Category != "terraform_dir" {
		t.Errorf("category = %q, want terraform_dir", got.Category)
	}
	if got.Safety != SafetySafe {
		t.Errorf("safety = %v, want SafetySafe", got.Safety)
	}
	if got.Strategy != StrategyTrash {
		t.Errorf("strategy = %v, want StrategyTrash", got.Strategy)
	}
}

func TestTerraformDirDetector_AcceptsJSONVariant(t *testing.T) {
	root := t.TempDir()
	stack := filepath.Join(root, "iac-json")
	if err := os.MkdirAll(filepath.Join(stack, ".terraform"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stack, "main.tf.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := terraformDirDetector{}
	if got := d.Match(context.Background(), filepath.Join(stack, ".terraform"),
		fakeEntry{name: ".terraform", isDir: true}); got == nil {
		t.Error("expected match when *.tf.json sibling exists")
	}
}

func TestTerraformDirDetector_RejectsNonDir(t *testing.T) {
	d := terraformDirDetector{}
	if got := d.Match(context.Background(), "/tmp/x/.terraform",
		fakeEntry{name: ".terraform", isDir: false}); got != nil {
		t.Error("expected no match for non-directory entry")
	}
	if got := d.Match(context.Background(), "/tmp/x/random",
		fakeEntry{name: "random", isDir: true}); got != nil {
		t.Error("expected no match for non-.terraform directory")
	}
}

func TestTerragruntCacheDetector_RequiresTerragruntHCL(t *testing.T) {
	d := Default.Get("terragrunt_cache")
	if d == nil {
		t.Fatal("terragrunt_cache detector not registered")
	}

	root := t.TempDir()
	stack := filepath.Join(root, "live")
	if err := os.MkdirAll(filepath.Join(stack, ".terragrunt-cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// No terragrunt.hcl → no match.
	if got := d.Match(ctx, filepath.Join(stack, ".terragrunt-cache"),
		fakeEntry{name: ".terragrunt-cache", isDir: true}); got != nil {
		t.Errorf("no match expected without terragrunt.hcl, got %+v", got)
	}

	// Add terragrunt.hcl → match.
	if err := os.WriteFile(filepath.Join(stack, "terragrunt.hcl"), []byte("include {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := d.Match(ctx, filepath.Join(stack, ".terragrunt-cache"),
		fakeEntry{name: ".terragrunt-cache", isDir: true}); got == nil {
		t.Error("expected match when terragrunt.hcl is present")
	}
}

func TestTerraformDetectors_RegisteredAndSafe(t *testing.T) {
	for _, id := range []Category{"terraform_dir", "terragrunt_cache"} {
		d := Default.Get(id)
		if d == nil {
			t.Errorf("%s not registered in Default registry", id)
			continue
		}
		if d.Safety() != SafetySafe {
			t.Errorf("%s safety = %v, want SafetySafe", id, d.Safety())
		}
		if d.DefaultStrategy() != StrategyTrash {
			t.Errorf("%s strategy = %v, want StrategyTrash", id, d.DefaultStrategy())
		}
		if !d.PlatformSupported() {
			t.Errorf("%s should be supported on every platform", id)
		}
		if !d.StopDescent() {
			t.Errorf("%s should stop descent into matched dir", id)
		}
	}
}

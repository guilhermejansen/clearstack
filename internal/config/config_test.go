package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Profile != "balanced" {
		t.Errorf("profile = %q, want balanced", cfg.Profile)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.Dormancy.MinAge != "14d" {
		t.Errorf("dormancy.min_age = %q, want 14d", cfg.Dormancy.MinAge)
	}
}

func TestDormancyParseMinAge(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"", 0, false},
		{"14d", 14 * 24 * time.Hour, false},
		{"3h", 3 * time.Hour, false},
		{"garbage", 0, true},
	}
	for _, c := range cases {
		got, err := Dormancy{MinAge: c.in}.ParseMinAge()
		if c.err && err == nil {
			t.Errorf("ParseMinAge(%q) expected error, got nil", c.in)
			continue
		}
		if !c.err && err != nil {
			t.Errorf("ParseMinAge(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseMinAge(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestApplyProfile(t *testing.T) {
	cases := []struct {
		name    string
		wantAge string
	}{
		{"balanced", "14d"},
		{"conservative", "30d"},
		{"aggressive", "7d"},
		{"fullstack", "14d"},
	}
	for _, c := range cases {
		cfg, err := ApplyProfile(Config{}, c.name)
		if err != nil {
			t.Errorf("ApplyProfile(%q): %v", c.name, err)
			continue
		}
		if cfg.Dormancy.MinAge != c.wantAge {
			t.Errorf("profile %q: dormancy = %q, want %q", c.name, cfg.Dormancy.MinAge, c.wantAge)
		}
	}
	if _, err := ApplyProfile(Config{}, "nope"); err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestSaveAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := &Config{
		Version:  1,
		Profile:  "aggressive",
		Dormancy: Dormancy{MinAge: "7d", CheckGit: true},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Profile != "aggressive" {
		t.Errorf("profile = %q, want aggressive", loaded.Profile)
	}
	if loaded.Dormancy.MinAge != "7d" {
		t.Errorf("min_age = %q, want 7d", loaded.Dormancy.MinAge)
	}
}

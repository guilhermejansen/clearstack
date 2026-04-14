package config

import "fmt"

// Profile names built into the binary.
const (
	ProfileConservative = "conservative"
	ProfileBalanced     = "balanced"
	ProfileAggressive   = "aggressive"
	ProfileFullstack    = "fullstack"
)

// ProfileNames returns every built-in profile identifier.
func ProfileNames() []string {
	return []string{ProfileConservative, ProfileBalanced, ProfileAggressive, ProfileFullstack}
}

// ApplyProfile returns a new Config derived from the caller's Config with
// the named profile's overrides applied.
//
// Profiles are intentionally conservative about enabling categories: they
// set defaults, users can always extend via config file or --categories.
func ApplyProfile(cfg Config, name string) (Config, error) {
	switch name {
	case "", ProfileBalanced:
		cfg.Profile = ProfileBalanced
		cfg.Dormancy.MinAge = "14d"
		cfg.Docker.Volumes = false
	case ProfileConservative:
		cfg.Profile = ProfileConservative
		cfg.Dormancy.MinAge = "30d"
		cfg.Safety.DefaultStrategy = "trash"
		cfg.Docker.Enabled = true
		cfg.Docker.Volumes = false
		cfg.Docker.BuildCache = false
	case ProfileAggressive:
		cfg.Profile = ProfileAggressive
		cfg.Dormancy.MinAge = "7d"
		cfg.Docker.Volumes = false // still opt-in; never auto
		cfg.Docker.BuildCache = true
	case ProfileFullstack:
		cfg.Profile = ProfileFullstack
		cfg.Dormancy.MinAge = "14d"
		cfg.Docker.Enabled = true
		cfg.Docker.BuildCache = true
	default:
		return cfg, fmt.Errorf("config: unknown profile %q", name)
	}
	return cfg, nil
}

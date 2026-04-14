// Package config loads, validates, and serializes clearstack configuration
// using viper. It also owns the built-in profile definitions.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/guilhermejansen/clearstack/internal/platform"
)

// Config is the root configuration schema loaded from ~/.config/clearstack/config.yaml
// (or the platform equivalent).
type Config struct {
	Version    int         `mapstructure:"version" yaml:"version"`
	Profile    string      `mapstructure:"profile" yaml:"profile"`
	Dormancy   Dormancy    `mapstructure:"dormancy" yaml:"dormancy"`
	Safety     SafetyBlock `mapstructure:"safety" yaml:"safety"`
	Categories Categories  `mapstructure:"categories" yaml:"categories"`
	Roots      []string    `mapstructure:"roots" yaml:"roots,omitempty"`
	Docker     Docker      `mapstructure:"docker" yaml:"docker"`
	UI         UI          `mapstructure:"ui" yaml:"ui"`
	Telemetry  Telemetry   `mapstructure:"telemetry" yaml:"telemetry"`
}

// Dormancy configures project-dormancy filtering.
type Dormancy struct {
	MinAge   string `mapstructure:"min_age" yaml:"min_age"`
	CheckGit bool   `mapstructure:"check_git" yaml:"check_git"`
}

// ParseMinAge returns the dormancy threshold as a duration. Supports the
// "14d" suffix in addition to time.ParseDuration units.
func (d Dormancy) ParseMinAge() (time.Duration, error) {
	s := strings.TrimSpace(d.MinAge)
	if s == "" {
		return 0, nil
	}
	if strings.HasSuffix(s, "d") {
		days, err := time.ParseDuration(strings.TrimSuffix(s, "d") + "h")
		if err != nil {
			return 0, fmt.Errorf("config: invalid dormancy.min_age %q: %w", s, err)
		}
		return days * 24, nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config: invalid dormancy.min_age %q: %w", s, err)
	}
	return dur, nil
}

// SafetyBlock exposes the user-facing whitelist hooks.
type SafetyBlock struct {
	DefaultStrategy       string   `mapstructure:"default_strategy" yaml:"default_strategy"`
	RequireDualConfirmFor []string `mapstructure:"require_dual_confirm_for" yaml:"require_dual_confirm_for"`
	WhitelistPaths        []string `mapstructure:"whitelist_paths" yaml:"whitelist_paths"`
}

// Categories lists enabled/disabled detectors by id.
type Categories struct {
	Enabled  []string `mapstructure:"enabled" yaml:"enabled,omitempty"`
	Disabled []string `mapstructure:"disabled" yaml:"disabled,omitempty"`
}

// Docker toggles Docker-related detectors.
type Docker struct {
	Enabled    bool `mapstructure:"enabled" yaml:"enabled"`
	Volumes    bool `mapstructure:"volumes" yaml:"volumes"`
	BuildCache bool `mapstructure:"build_cache" yaml:"build_cache"`
}

// UI holds TUI preferences.
type UI struct {
	Theme         string `mapstructure:"theme" yaml:"theme"`
	DefaultSort   string `mapstructure:"default_sort" yaml:"default_sort"`
	DefaultFilter string `mapstructure:"default_filter" yaml:"default_filter"`
}

// Telemetry is opt-in only.
type Telemetry struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

// DefaultConfigPath returns the path clearstack loads config from when no
// --config flag is supplied.
func DefaultConfigPath() string {
	return filepath.Join(platform.ConfigDir(), "config.yaml")
}

// Load reads the config file pointed at by path (or DefaultConfigPath when
// empty). When the file does not exist, Load returns a Config with defaults
// applied so callers never need to nil-check.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	ApplyDefaults(v)
	if err := v.ReadInConfig(); err != nil {
		var nfErr viper.ConfigFileNotFoundError
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || isNotExist(err) || os.IsNotExist(err) {
			// Missing file: return the defaults snapshot.
			return fromViper(v)
		}
		_ = nfErr
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	return fromViper(v)
}

// Save writes cfg out as YAML to path.
func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}
	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("version", cfg.Version)
	v.Set("profile", cfg.Profile)
	v.Set("dormancy", cfg.Dormancy)
	v.Set("safety", cfg.Safety)
	v.Set("categories", cfg.Categories)
	v.Set("roots", cfg.Roots)
	v.Set("docker", cfg.Docker)
	v.Set("ui", cfg.UI)
	v.Set("telemetry", cfg.Telemetry)
	return v.WriteConfigAs(path)
}

func fromViper(v *viper.Viper) (*Config, error) {
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	if c.Profile == "" {
		c.Profile = "balanced"
	}
	if c.Version == 0 {
		c.Version = 1
	}
	return &c, nil
}

func isNotExist(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such file")
}

package config

import "github.com/spf13/viper"

// ApplyDefaults registers the built-in default values on a viper instance.
// Callers can later override via a config file, env vars, or flags.
func ApplyDefaults(v *viper.Viper) {
	v.SetDefault("version", 1)
	v.SetDefault("profile", "balanced")
	v.SetDefault("dormancy.min_age", "14d")
	v.SetDefault("dormancy.check_git", true)

	v.SetDefault("safety.default_strategy", "trash")
	v.SetDefault("safety.require_dual_confirm_for", []string{
		"pnpm_store_raw", "m2_repository", "avd", "docker_volumes",
	})
	v.SetDefault("safety.whitelist_paths", []string{})

	v.SetDefault("categories.enabled", []string{})
	v.SetDefault("categories.disabled", []string{})

	v.SetDefault("roots", []string{})

	v.SetDefault("docker.enabled", true)
	v.SetDefault("docker.volumes", false)
	v.SetDefault("docker.build_cache", true)

	v.SetDefault("ui.theme", "auto")
	v.SetDefault("ui.default_sort", "size")
	v.SetDefault("ui.default_filter", "")

	v.SetDefault("telemetry.enabled", false)
}

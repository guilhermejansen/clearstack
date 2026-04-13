// Package platform exposes OS detection and well-known path helpers.
package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// OS identifiers.
const (
	Darwin  = "darwin"
	Linux   = "linux"
	Windows = "windows"
)

// Current returns runtime.GOOS.
func Current() string { return runtime.GOOS }

// IsDarwin reports whether we're running on macOS.
func IsDarwin() bool { return runtime.GOOS == Darwin }

// IsLinux reports whether we're running on Linux.
func IsLinux() bool { return runtime.GOOS == Linux }

// IsWindows reports whether we're running on Windows.
func IsWindows() bool { return runtime.GOOS == Windows }

// Home returns the user's home directory, or empty string on failure.
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	h := Home()
	if h == "" {
		return p
	}
	if p == "~" {
		return h
	}
	if strings.HasPrefix(p, "~/") || (IsWindows() && strings.HasPrefix(p, `~\`)) {
		return filepath.Join(h, p[2:])
	}
	return p
}

// StateDir returns the platform state directory for clearstack
// (for logs, journals, etc.).
//
// macOS:  ~/Library/Application Support/clearstack
// Linux:  $XDG_STATE_HOME/clearstack or ~/.local/state/clearstack
// Win:    %LOCALAPPDATA%\clearstack\state
func StateDir() string {
	switch Current() {
	case Darwin:
		return filepath.Join(Home(), "Library", "Application Support", "clearstack")
	case Windows:
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "clearstack", "state")
		}
		return filepath.Join(Home(), "AppData", "Local", "clearstack", "state")
	default:
		if v := os.Getenv("XDG_STATE_HOME"); v != "" {
			return filepath.Join(v, "clearstack")
		}
		return filepath.Join(Home(), ".local", "state", "clearstack")
	}
}

// ConfigDir returns the platform config directory for clearstack.
//
// macOS:  ~/Library/Application Support/clearstack
// Linux:  $XDG_CONFIG_HOME/clearstack or ~/.config/clearstack
// Win:    %APPDATA%\clearstack
func ConfigDir() string {
	switch Current() {
	case Darwin:
		return filepath.Join(Home(), "Library", "Application Support", "clearstack")
	case Windows:
		if v := os.Getenv("APPDATA"); v != "" {
			return filepath.Join(v, "clearstack")
		}
		return filepath.Join(Home(), "AppData", "Roaming", "clearstack")
	default:
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return filepath.Join(v, "clearstack")
		}
		return filepath.Join(Home(), ".config", "clearstack")
	}
}

// CacheDir returns the platform cache directory for clearstack.
func CacheDir() string {
	switch Current() {
	case Darwin:
		return filepath.Join(Home(), "Library", "Caches", "clearstack")
	case Windows:
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "clearstack", "cache")
		}
		return filepath.Join(Home(), "AppData", "Local", "clearstack", "cache")
	default:
		if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
			return filepath.Join(v, "clearstack")
		}
		return filepath.Join(Home(), ".cache", "clearstack")
	}
}

// PathEqual compares two paths, case-insensitive on Windows.
func PathEqual(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if IsWindows() {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// PathHasPrefix reports whether path is at or below prefix,
// using platform-appropriate comparison.
func PathHasPrefix(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	if IsWindows() {
		path = strings.ToLower(path)
		prefix = strings.ToLower(prefix)
	}
	if path == prefix {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(path, prefix)
}

// Package version exposes build-time version metadata injected via -ldflags.
package version

import "fmt"

// Build metadata — overridden at link time by GoReleaser / Makefile.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Full returns a human-readable version string "vX.Y.Z (commit, date)".
func Full() string {
	return fmt.Sprintf("%s (%s, %s)", Version, Commit, Date)
}

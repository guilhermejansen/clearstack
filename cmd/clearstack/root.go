package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/version"
)

// globalFlags holds persistent flag values shared by every subcommand.
var globalFlags struct {
	ConfigPath string
	Profile    string
	Verbose    bool
	Quiet      bool
	NoColor    bool
	JSON       bool
	Yes        bool
	DryRun     bool
	LogLevel   string
}

// newRootCmd wires up every subcommand and returns the root.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "clearstack",
		Short: "Safe cross-platform developer disk cleanup for every stack",
		Long: `clearstack scans your machine for dormant build artifacts, caches, and
package-manager stores, then safely reclaims disk space without breaking any
active project.

By default it opens an interactive TUI. Use subcommands (scan, clean,
analyze, doctor, config, undo) for scriptable workflows.`,
		Version:       version.Full(),
		SilenceUsage:  true,
		SilenceErrors: true,
		// When invoked with no subcommand, fall through to TUI (Sprint 4).
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUIPlaceholder(cmd.OutOrStdout())
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&globalFlags.ConfigPath, "config", "", "path to config file (default: platform config dir)")
	pf.StringVar(&globalFlags.Profile, "profile", "", "profile to use (conservative | balanced | aggressive | fullstack)")
	pf.BoolVarP(&globalFlags.Verbose, "verbose", "v", false, "verbose output")
	pf.BoolVarP(&globalFlags.Quiet, "quiet", "q", false, "suppress non-error output")
	pf.BoolVar(&globalFlags.NoColor, "no-color", false, "disable ANSI colors")
	pf.BoolVar(&globalFlags.JSON, "json", false, "emit JSON instead of human text")
	pf.BoolVarP(&globalFlags.Yes, "yes", "y", false, "skip interactive confirmations")
	pf.BoolVar(&globalFlags.DryRun, "dry-run", false, "report intended actions without touching anything")
	pf.StringVar(&globalFlags.LogLevel, "log-level", "info", "log level (debug|info|warn|error)")

	root.AddCommand(
		newVersionCmd(),
		newScanCmd(),
		newCleanCmd(),
		newAnalyzeCmd(),
		newDoctorCmd(),
		newConfigCmd(),
		newUndoCmd(),
		newCategoriesCmd(),
		newCompletionCmd(root),
	)
	return root
}

// Execute runs the cobra tree and exits the process with the appropriate code.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

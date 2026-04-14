package main

import (
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/config"
	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/platform"
	"github.com/guilhermejansen/clearstack/internal/trash"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose clearstack's environment and detector availability",
		RunE:  runDoctor,
	}
}

type doctorReport struct {
	OS         string            `json:"os"`
	ConfigPath string            `json:"config_path"`
	StateDir   string            `json:"state_dir"`
	CacheDir   string            `json:"cache_dir"`
	Trasher    string            `json:"trasher"`
	Tools      map[string]bool   `json:"tools"`
	Detectors  []detectorSummary `json:"detectors"`
	Docker     map[string]any    `json:"docker,omitempty"`
}

type detectorSummary struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Safety      string `json:"safety"`
	Strategy    string `json:"strategy"`
	Supported   bool   `json:"supported"`
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	tools := map[string]bool{}
	for _, name := range []string{"go", "npm", "yarn", "pnpm", "bun", "pip", "poetry", "uv", "cargo", "pod", "dotnet", "docker", "git"} {
		_, err := exec.LookPath(name)
		tools[name] = err == nil
	}

	var ds []detectorSummary
	for _, d := range detectors.Default.All() {
		ds = append(ds, detectorSummary{
			ID:          string(d.ID()),
			Description: d.Description(),
			Safety:      d.Safety().String(),
			Strategy:    string(d.DefaultStrategy()),
			Supported:   d.PlatformSupported(),
		})
	}

	r := doctorReport{
		OS:         platform.Current(),
		ConfigPath: config.DefaultConfigPath(),
		StateDir:   platform.StateDir(),
		CacheDir:   platform.CacheDir(),
		Trasher:    trash.New().Name(),
		Tools:      tools,
		Detectors:  ds,
	}
	if tools["docker"] {
		if df, err := detectors.DockerDF(cmd.Context()); err == nil {
			r.Docker = df
		}
	}

	if globalFlags.JSON {
		return writeJSON(cmd.OutOrStdout(), r)
	}

	printfln(cmd.OutOrStdout(), "clearstack doctor")
	printfln(cmd.OutOrStdout(), "  os           : %s", r.OS)
	printfln(cmd.OutOrStdout(), "  config       : %s", r.ConfigPath)
	printfln(cmd.OutOrStdout(), "  state dir    : %s", r.StateDir)
	printfln(cmd.OutOrStdout(), "  cache dir    : %s", r.CacheDir)
	printfln(cmd.OutOrStdout(), "  trasher      : %s", r.Trasher)
	printfln(cmd.OutOrStdout(), "\ntools on PATH:")
	for name, ok := range r.Tools {
		printfln(cmd.OutOrStdout(), "  %-8s %s", name, yesno(ok))
	}
	if len(r.Docker) > 0 {
		printfln(cmd.OutOrStdout(), "\ndocker system df:")
		for k, v := range r.Docker {
			printfln(cmd.OutOrStdout(), "  %-16s %v", k, v)
		}
	}
	printfln(cmd.OutOrStdout(), "\ndetectors (%d):", len(r.Detectors))
	for _, d := range r.Detectors {
		printfln(cmd.OutOrStdout(), "  %-22s [%s] supported=%s  %s", d.ID, d.Safety, yesno(d.Supported), d.Description)
	}
	return nil
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no "
}

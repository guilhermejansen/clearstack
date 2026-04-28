package detectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Docker detectors wrap the official `docker` CLI. We intentionally avoid
// importing github.com/docker/docker/client to keep the clearstack binary
// small and CGO-free; the CLI already provides every prune subcommand we
// need and is the canonical way to reach both local Docker Desktop and
// remote contexts.
//
// A missing `docker` binary or unreachable daemon is handled gracefully:
// PlatformSupported() returns false, doctor surfaces the state, and every
// detector skips itself.

// dockerAvailable performs an inexpensive probe of the daemon.
func dockerAvailable() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	// `docker info` talks to the daemon; `docker version --format` does not
	// require the daemon when it fails fast.
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// DockerDF returns a best-effort parse of `docker system df --format=json`.
// It is used by `clearstack doctor` and `clearstack analyze` to report
// reclaimable space. Errors are swallowed by callers.
func DockerDF(ctx context.Context) (map[string]any, error) {
	cmd := exec.CommandContext(ctx, "docker", "system", "df", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker system df: %w", err)
	}
	result := make(map[string]any)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if t, ok := m["Type"].(string); ok {
			result[t] = m
		}
	}
	return result, nil
}

// dockerReclaimedRE matches the reclaimed-bytes trailer Docker emits at the
// end of every prune subcommand. Two phrasings exist in the wild:
//
//   - `Total reclaimed space: X.YGB` (image / container / volume prune)
//   - `Total:\tX.YGB`                (builder prune since 24.x)
//
// The unit suffix is Docker's short IEC form (B / KB / MB / GB / TB) with
// or without a leading decimal (`1.2GB`, `0B`, `512MB`).
var dockerReclaimedRE = regexp.MustCompile(`(?i)(?:Total reclaimed space|Total):\s*([0-9]+(?:\.[0-9]+)?)\s*([KMGT]?B)`)

// parseDockerReclaimed parses Docker's "Total reclaimed space: X.YGB" line
// from the combined output of any `docker prune` subcommand and returns
// the byte count. Unmatched output → 0 (the Cleaner falls back to 0 too,
// since pseudo matches have no SizeBytes).
func parseDockerReclaimed(out string) int64 {
	m := dockerReclaimedRE.FindStringSubmatch(out)
	if len(m) != 3 {
		return 0
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil || n < 0 {
		return 0
	}
	mult := int64(1)
	switch strings.ToUpper(m[2]) {
	case "B":
		mult = 1
	case "KB":
		mult = 1000
	case "MB":
		mult = 1000 * 1000
	case "GB":
		mult = 1000 * 1000 * 1000
	case "TB":
		mult = 1000 * 1000 * 1000 * 1000
	}
	if n > float64(int64(1)<<62)/float64(mult) {
		return 0
	}
	return int64(n * float64(mult))
}

// baseDocker provides the shared boilerplate for every Docker detector.
type baseDocker struct {
	id   Category
	desc string
	argv []string
	safe Safety
}

func (b baseDocker) ID() Category              { return b.id }
func (b baseDocker) Description() string       { return b.desc }
func (b baseDocker) Safety() Safety            { return b.safe }
func (b baseDocker) DefaultStrategy() Strategy { return StrategyNativeCommand }
func (b baseDocker) RequiresDormancy() bool    { return false }
func (b baseDocker) StopDescent() bool         { return true }
func (b baseDocker) PlatformSupported() bool   { return dockerAvailable() }
func (b baseDocker) NativeCommand(_ Match) ([]string, string) {
	return b.argv, "y\n"
}

// ParseNativeOutput satisfies the NativeOutputParser interface for every
// Docker detector. It surfaces real reclaimed bytes in the run summary
// instead of the misleading "0 B" the Cleaner would otherwise report for
// pseudo matches with no fs footprint.
func (b baseDocker) ParseNativeOutput(out string) int64 {
	return parseDockerReclaimed(out)
}

// Match for Docker detectors: the scanner cannot find Docker state on disk,
// so all Docker detectors return nil from Match() (scan-inert). They emit a
// synthetic Match through the Synthesizer interface when the user explicitly
// asks for the category via `clearstack clean --categories=docker_*`. The
// synthetic Path uses the form "docker:<kind>" which trips Match.IsPseudo()
// and tells the Cleaner to skip filesystem-safety validation.

// synthesize is the shared implementation for every Docker detector's
// Synthesize() method. It returns nil when the daemon is unreachable so the
// command layer can report a friendly "docker daemon unavailable" instead
// of running a doomed prune.
func (b baseDocker) synthesize() *Match {
	if !dockerAvailable() {
		return nil
	}
	return &Match{
		Path:     "docker:" + string(b.id),
		Category: b.id,
		Safety:   b.safe,
		Strategy: StrategyNativeCommand,
	}
}

type dockerImages struct{ baseDocker }

func (*dockerImages) Match(_ context.Context, _ string, _ fs.DirEntry) *Match { return nil }
func (d *dockerImages) Synthesize() *Match                                    { return d.baseDocker.synthesize() }

func newDockerImages() *dockerImages {
	return &dockerImages{baseDocker: baseDocker{
		id:   "docker_images",
		desc: "Dangling Docker images (docker image prune -f --filter dangling=true)",
		argv: []string{"docker", "image", "prune", "-f", "--filter", "dangling=true"},
		safe: SafetySafe,
	}}
}

type dockerContainers struct{ baseDocker }

func (*dockerContainers) Match(_ context.Context, _ string, _ fs.DirEntry) *Match { return nil }
func (d *dockerContainers) Synthesize() *Match                                    { return d.baseDocker.synthesize() }

func newDockerContainers() *dockerContainers {
	return &dockerContainers{baseDocker: baseDocker{
		id:   "docker_containers",
		desc: "Stopped Docker containers (docker container prune -f)",
		argv: []string{"docker", "container", "prune", "-f"},
		safe: SafetySafe,
	}}
}

type dockerBuildCache struct{ baseDocker }

func (*dockerBuildCache) Match(_ context.Context, _ string, _ fs.DirEntry) *Match { return nil }
func (d *dockerBuildCache) Synthesize() *Match                                    { return d.baseDocker.synthesize() }

func newDockerBuildCache() *dockerBuildCache {
	return &dockerBuildCache{baseDocker: baseDocker{
		id:   "docker_build_cache",
		desc: "Docker builder cache (docker builder prune -f)",
		argv: []string{"docker", "builder", "prune", "-f"},
		safe: SafetySafe,
	}}
}

type dockerNetworks struct{ baseDocker }

func (*dockerNetworks) Match(_ context.Context, _ string, _ fs.DirEntry) *Match { return nil }
func (d *dockerNetworks) Synthesize() *Match                                    { return d.baseDocker.synthesize() }

func newDockerNetworks() *dockerNetworks {
	return &dockerNetworks{baseDocker: baseDocker{
		id:   "docker_networks",
		desc: "Unused Docker networks (docker network prune -f)",
		argv: []string{"docker", "network", "prune", "-f"},
		safe: SafetySafe,
	}}
}

// dockerVolumes is intentionally more guarded — volumes can hold real data.
// It is registered with SafetyDanger and requires explicit --categories=docker_volumes
// AND the `--yes` flag to run.
type dockerVolumes struct{ baseDocker }

func (*dockerVolumes) Match(_ context.Context, _ string, _ fs.DirEntry) *Match { return nil }
func (d *dockerVolumes) Synthesize() *Match                                    { return d.baseDocker.synthesize() }

func newDockerVolumes() *dockerVolumes {
	return &dockerVolumes{baseDocker: baseDocker{
		id:   "docker_volumes",
		desc: "Unused Docker volumes (docker volume prune -f) — DANGER: holds data",
		argv: []string{"docker", "volume", "prune", "-f"},
		safe: SafetyDanger,
	}}
}

// Override the top-level init() list: we use factory functions so the
// concrete types carry their baseDocker values correctly.
func init() {
	Register(newDockerImages())
	Register(newDockerContainers())
	Register(newDockerBuildCache())
	Register(newDockerNetworks())
	Register(newDockerVolumes())
}

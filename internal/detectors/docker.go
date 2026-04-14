package detectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
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

// Match for Docker detectors: the scanner never needs to walk to find
// Docker state; we emit a synthetic Match with Path="docker://<kind>" when
// the engine explicitly runs the Docker category. Since there is no real
// path to match on disk, the scanner normally won't trigger these — the
// `clean --categories=docker_*` path invokes them directly via the registry.
//
// To still allow scans to report what's reclaimable, we return a synthetic
// match when the very first visited directory matches; the command layer
// then filters by category.
type dockerImages struct{ baseDocker }

func (*dockerImages) Match(_ context.Context, _ string, _ fs.DirEntry) *Match {
	// Docker detectors are intentionally scan-inert — they're triggered
	// explicitly from the clean command. Returning nil here keeps the
	// scanner clean and still allows direct `clearstack clean --categories=docker_images`.
	return nil
}

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

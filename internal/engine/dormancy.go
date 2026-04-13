package engine

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DormancyPolicy filters matches by the mtime of the containing project.
//
// The zero value disables dormancy filtering entirely.
type DormancyPolicy struct {
	// MinAge is the minimum time since the last modification for a match to
	// count as dormant. Zero disables the check.
	MinAge time.Duration
	// CheckGit, when true and a .git directory is present at or above the
	// match, consults `git log -1 --format=%ct` to refine the timestamp.
	CheckGit bool
	// Clock allows tests to inject time.
	Clock func() time.Time
}

// Default returns a policy with a 14-day threshold and git enabled.
func DefaultDormancy() DormancyPolicy {
	return DormancyPolicy{
		MinAge:   14 * 24 * time.Hour,
		CheckGit: true,
		Clock:    time.Now,
	}
}

// IsDormant reports whether path's effective modification time is older than
// the configured threshold. When MinAge <= 0 it always returns true.
func (p DormancyPolicy) IsDormant(ctx context.Context, path string, fallback time.Time) bool {
	if p.MinAge <= 0 {
		return true
	}
	now := p.clock()
	t := fallback
	if p.CheckGit {
		if gt, ok := gitLastCommit(ctx, path); ok && gt.After(t) {
			t = gt
		}
	}
	if t.IsZero() {
		return true
	}
	return now.Sub(t) >= p.MinAge
}

func (p DormancyPolicy) clock() time.Time {
	if p.Clock != nil {
		return p.Clock()
	}
	return time.Now()
}

// gitLastCommit walks upward from path looking for a .git directory; if one is
// found, it returns the timestamp of the most recent commit on HEAD.
func gitLastCommit(ctx context.Context, path string) (time.Time, bool) {
	repo := findGitRoot(path)
	if repo == "" {
		return time.Time{}, false
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "log", "-1", "--format=%ct")
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	secStr := strings.TrimSpace(string(out))
	if secStr == "" {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, 0), true
}

func findGitRoot(path string) string {
	cur := filepath.Clean(path)
	for i := 0; i < 32; i++ {
		candidate := filepath.Join(cur, ".git")
		if fi, err := statCached(candidate); err == nil && fi != nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
	return ""
}

package engine

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/charlievieth/fastwalk"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/platform"
)

// Scanner walks filesystem roots in parallel with fastwalk and emits matches
// classified by a Classifier.
//
// Safety: Scanner never follows symlinks. Attempting to configure it
// otherwise is intentionally unsupported — escape-via-symlink is a documented
// class of bug in similar tools and clearstack refuses to accept the risk.
type Scanner struct {
	Classifier *Classifier
	Safety     *Safety
	NumWorkers int
}

// NewScanner constructs a Scanner with the given classifier and safety check.
// When workers <= 0 the scanner defaults to runtime.NumCPU().
func NewScanner(c *Classifier, s *Safety, workers int) *Scanner {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Scanner{Classifier: c, Safety: s, NumWorkers: workers}
}

// Scan walks each root concurrently, emitting matches on out until every root
// has been visited or ctx is canceled. It returns a joined error, if any.
//
// Scan closes out before returning to signal completion to consumers.
func (s *Scanner) Scan(ctx context.Context, roots []string, out chan<- detectors.Match) error {
	defer close(out)
	if len(roots) == 0 {
		return errors.New("scanner: no roots provided")
	}
	if s.Classifier == nil {
		return errors.New("scanner: classifier is nil")
	}

	conf := &fastwalk.Config{
		Follow:     false,
		NumWorkers: s.NumWorkers,
		Sort:       fastwalk.SortFilesFirst,
	}

	var (
		errMu sync.Mutex
		errs  []error
	)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		errs = append(errs, err)
		errMu.Unlock()
	}

	for _, raw := range roots {
		if ctx.Err() != nil {
			break
		}
		root := filepath.Clean(platform.ExpandHome(raw))
		if root == "" {
			continue
		}

		walkFn := func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Permission errors etc. — skip the dir, keep scanning.
				if d != nil && d.IsDir() {
					return fastwalk.SkipDir
				}
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Never descend into protected roots.
			if d.IsDir() && s.Safety != nil {
				// Only hard-refuse when the path *is* protected; subpaths
				// are allowed because many detectors live inside them.
				if s.Safety.Validate(path) != nil && platform.PathEqual(path, root) {
					return fastwalk.SkipDir
				}
			}
			m, stopDescent := s.Classifier.Match(ctx, path, d)
			if m == nil {
				return nil
			}
			// Populate mtime lazily from DirEntry Info; ignore errors.
			if info, infoErr := d.Info(); infoErr == nil && m.ModTime.IsZero() {
				m.ModTime = info.ModTime()
			}
			m.Path = filepath.Clean(m.Path)
			if m.Path == "" {
				m.Path = filepath.Clean(path)
			}
			select {
			case out <- *m:
			case <-ctx.Done():
				return ctx.Err()
			}
			if stopDescent && d.IsDir() {
				return fastwalk.SkipDir
			}
			return nil
		}

		if err := fastwalk.Walk(conf, root, walkFn); err != nil && !errors.Is(err, context.Canceled) {
			recordErr(err)
		}
	}
	return errors.Join(errs...)
}

package engine

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/charlievieth/fastwalk"
)

// Sizer computes the on-disk byte size of a filesystem subtree.
//
// It uses fastwalk for speed and does not follow symlinks (identical
// safety stance to Scanner).
type Sizer struct {
	NumWorkers int
}

// NewSizer returns a Sizer. workers <= 0 defaults to runtime.NumCPU().
func NewSizer(workers int) *Sizer {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Sizer{NumWorkers: workers}
}

// Size returns the recursive byte size of path.
//
// Errors reading individual entries are swallowed — the returned size is a
// best-effort lower bound on the space the user could reclaim.
func (s *Sizer) Size(ctx context.Context, path string) (int64, error) {
	if path == "" {
		return 0, errors.New("sizer: empty path")
	}
	path = filepath.Clean(path)

	var total int64
	conf := &fastwalk.Config{
		Follow:     false,
		NumWorkers: s.NumWorkers,
	}
	walkFn := func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fastwalk.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		atomic.AddInt64(&total, info.Size())
		return nil
	}
	if err := fastwalk.Walk(conf, path, walkFn); err != nil && !errors.Is(err, context.Canceled) {
		return atomic.LoadInt64(&total), err
	}
	return atomic.LoadInt64(&total), nil
}

// SizeMany resolves sizes for a batch of paths concurrently, calling cb for
// each result as it completes. The callback is invoked from multiple
// goroutines.
func (s *Sizer) SizeMany(ctx context.Context, paths []string, cb func(path string, size int64, err error)) {
	if cb == nil {
		return
	}
	sem := make(chan struct{}, s.NumWorkers)
	var wg sync.WaitGroup
	for _, p := range paths {
		if ctx.Err() != nil {
			break
		}
		p := p
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			n, err := s.Size(ctx, p)
			cb(p, n, err)
		}()
	}
	wg.Wait()
}

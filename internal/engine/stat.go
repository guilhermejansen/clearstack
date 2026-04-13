package engine

import (
	"io/fs"
	"os"
	"sync"
)

// statCached is a tiny process-wide memoized os.Lstat wrapper used by
// helpers that repeatedly stat the same ancestor directories during a scan
// (e.g., walking up looking for .git roots, lockfiles).
//
// Entries are never evicted; the cache is bounded by the number of unique
// paths visited during a single process lifetime and the memory cost is
// negligible compared to the savings.
var statCache sync.Map

type statEntry struct {
	fi  fs.FileInfo
	err error
}

func statCached(path string) (fs.FileInfo, error) {
	if v, ok := statCache.Load(path); ok {
		e := v.(statEntry)
		return e.fi, e.err
	}
	fi, err := os.Lstat(path)
	statCache.Store(path, statEntry{fi: fi, err: err})
	return fi, err
}

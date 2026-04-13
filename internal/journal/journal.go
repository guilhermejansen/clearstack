// Package journal records every clean operation in an append-only JSON Lines
// log for audit, undo, and debugging.
package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guilhermejansen/clearstack/internal/platform"
)

// Entry is one record in the journal.
type Entry struct {
	ID            string    `json:"id"`
	At            time.Time `json:"at"`
	Category      string    `json:"category"`
	Strategy      string    `json:"strategy"`
	OriginalPath  string    `json:"original_path"`
	TrashLocation string    `json:"trash_location,omitempty"`
	BytesFreed    int64     `json:"bytes_freed"`
	DryRun        bool      `json:"dry_run"`
	Undoable      bool      `json:"undoable"`
	Err           string    `json:"err,omitempty"`
	OS            string    `json:"os"`
	Host          string    `json:"host,omitempty"`
}

// Journal is a thread-safe append-only writer around operations.jsonl.
type Journal struct {
	mu   sync.Mutex
	path string
	f    *os.File
	w    *bufio.Writer
}

// Open creates (or opens for append) the journal file at the platform's state
// directory. Callers must Close when done.
func Open() (*Journal, error) {
	dir := platform.StateDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("journal: mkdir: %w", err)
	}
	path := filepath.Join(dir, "operations.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("journal: open: %w", err)
	}
	return &Journal{path: path, f: f, w: bufio.NewWriter(f)}, nil
}

// Path returns the on-disk location of the journal.
func (j *Journal) Path() string { return j.path }

// Append writes a single entry to the journal and flushes it immediately to
// disk. Returns an error when the underlying write fails.
func (j *Journal) Append(e Entry) error {
	if j == nil {
		return errors.New("journal: nil receiver")
	}
	if e.ID == "" {
		e.ID = generateID()
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if e.OS == "" {
		e.OS = platform.Current()
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.w == nil {
		return errors.New("journal: closed")
	}
	enc := json.NewEncoder(j.w)
	if err := enc.Encode(e); err != nil {
		return fmt.Errorf("journal: encode: %w", err)
	}
	if err := j.w.Flush(); err != nil {
		return fmt.Errorf("journal: flush: %w", err)
	}
	return nil
}

// Close flushes and closes the journal file.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.w != nil {
		if err := j.w.Flush(); err != nil {
			return fmt.Errorf("journal: flush on close: %w", err)
		}
		j.w = nil
	}
	if j.f != nil {
		if err := j.f.Close(); err != nil {
			return fmt.Errorf("journal: close: %w", err)
		}
		j.f = nil
	}
	return nil
}

// Read loads every entry from the journal file (newest last).
// It is safe to call while the journal is being written.
func Read(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("journal: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // tolerate partial writes
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("journal: scan: %w", err)
	}
	return entries, nil
}

func generateID() string {
	// 16-byte hex id keyed on time + pid. Not crypto — just collision-free.
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

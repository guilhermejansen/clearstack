package engine

import (
	"context"
	"io/fs"
	"time"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

// Classifier applies an ordered list of Detectors to filesystem entries
// emitted by the scanner and returns the first matching Match, if any.
//
// Classifier is stateless beyond the detector slice and safe for concurrent
// reads.
type Classifier struct {
	detectors []detectors.Detector
	stopIDs   map[detectors.Category]bool
}

// NewClassifier builds a Classifier over the given detectors. The order is
// preserved — earlier detectors win on ambiguous paths.
func NewClassifier(ds []detectors.Detector) *Classifier {
	stop := make(map[detectors.Category]bool, len(ds))
	for _, d := range ds {
		stop[d.ID()] = d.StopDescent()
	}
	return &Classifier{detectors: ds, stopIDs: stop}
}

// Detectors returns the classifier's detectors in order (read-only copy).
func (c *Classifier) Detectors() []detectors.Detector {
	out := make([]detectors.Detector, len(c.detectors))
	copy(out, c.detectors)
	return out
}

// Match runs each detector against the entry and returns the first non-nil
// Match. The second return reports whether the scanner should stop descending
// into the matched path (i.e., StopDescent() was true on the winning detector).
func (c *Classifier) Match(ctx context.Context, path string, entry fs.DirEntry) (*detectors.Match, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	for _, d := range c.detectors {
		m := d.Match(ctx, path, entry)
		if m == nil {
			continue
		}
		// Fill convenience fields the scanner did not.
		if m.FoundAt.IsZero() {
			m.FoundAt = time.Now()
		}
		if m.DetectorID == "" {
			m.DetectorID = string(d.ID())
		}
		if m.Category == "" {
			m.Category = d.ID()
		}
		if m.Safety == 0 {
			m.Safety = d.Safety()
		}
		if m.Strategy == "" {
			m.Strategy = d.DefaultStrategy()
		}
		return m, c.stopIDs[d.ID()]
	}
	return nil, false
}

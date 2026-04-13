package detectors

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Xcode DerivedData lives at ~/Library/Developer/Xcode/DerivedData and is
// often the single biggest reclaimable directory on a developer's Mac
// (30–100+ GB is routine).
//
// We register it as a singleton detector so the scanner does not need to
// walk into Library manually — the engine prepends this path to the scan
// root list when the detector is enabled.

func init() {
	if runtime.GOOS != "darwin" {
		return
	}
	Register(&xcodeDerivedData{})
}

type xcodeDerivedData struct {
	once sync.Once
	path string
	ok   bool
}

func (x *xcodeDerivedData) ID() Category { return "xcode_derived_data" }
func (x *xcodeDerivedData) Description() string {
	return "Xcode DerivedData (~/Library/Developer/Xcode/DerivedData)"
}
func (x *xcodeDerivedData) Safety() Safety            { return SafetySafe }
func (x *xcodeDerivedData) DefaultStrategy() Strategy { return StrategyTrash }
func (x *xcodeDerivedData) PlatformSupported() bool   { return runtime.GOOS == "darwin" }
func (x *xcodeDerivedData) RequiresDormancy() bool    { return false }
func (x *xcodeDerivedData) StopDescent() bool         { return true }

func (x *xcodeDerivedData) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() {
		return nil
	}
	x.once.Do(x.resolve)
	if !x.ok {
		return nil
	}
	if path != x.path {
		return nil
	}
	return &Match{
		Path:     path,
		Category: x.ID(),
		Safety:   SafetySafe,
		Strategy: StrategyTrash,
	}
}

func (x *xcodeDerivedData) resolve() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	p := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	if _, err := os.Lstat(p); err != nil {
		return
	}
	x.path = p
	x.ok = true
}

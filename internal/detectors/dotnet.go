package detectors

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
)

// .NET / C# build artifacts.
//
// `bin/` and `obj/` are regenerable but also common names for unrelated
// directories; we emit matches only when a sibling *.csproj / *.fsproj /
// *.vbproj exists, which is the canonical .NET signature.

func init() {
	Register(&dotnetDirDetector{id: "dotnet_bin", dir: "bin", desc: ".NET build output (bin/)"})
	Register(&dotnetDirDetector{id: "dotnet_obj", dir: "obj", desc: ".NET intermediate output (obj/)"})
}

type dotnetDirDetector struct {
	id   Category
	dir  string
	desc string
}

func (d *dotnetDirDetector) ID() Category              { return d.id }
func (d *dotnetDirDetector) Description() string       { return d.desc }
func (d *dotnetDirDetector) Safety() Safety            { return SafetySafe }
func (d *dotnetDirDetector) DefaultStrategy() Strategy { return StrategyTrash }
func (*dotnetDirDetector) PlatformSupported() bool     { return true }
func (*dotnetDirDetector) RequiresDormancy() bool      { return true }
func (*dotnetDirDetector) StopDescent() bool           { return true }

func (d *dotnetDirDetector) Match(_ context.Context, path string, entry fs.DirEntry) *Match {
	if entry == nil || !entry.IsDir() || entry.Name() != d.dir {
		return nil
	}
	parent := filepath.Dir(path)
	siblings, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	for _, s := range siblings {
		name := s.Name()
		switch filepath.Ext(name) {
		case ".csproj", ".fsproj", ".vbproj":
			return &Match{
				Path:     path,
				Category: d.id,
				Safety:   SafetySafe,
				Strategy: StrategyTrash,
			}
		}
	}
	return nil
}

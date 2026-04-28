package detectors

import (
	"io/fs"
	"os"
)

// osLstat is a thin indirection around os.Lstat used by helpers.go so tests
// can swap the implementation without touching real disk.
var osLstat func(name string) (fs.FileInfo, error) = os.Lstat

// osReadDir is a thin indirection around os.ReadDir used by detectors that
// need to inspect sibling files (e.g., terraform_dir checks for *.tf next to
// .terraform/). Tests can swap this for an in-memory fixture.
var osReadDir func(name string) ([]fs.DirEntry, error) = os.ReadDir

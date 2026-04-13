package detectors

import (
	"io/fs"
	"os"
)

// osLstat is a thin indirection around os.Lstat used by helpers.go so tests
// can swap the implementation without touching real disk.
var osLstat func(name string) (fs.FileInfo, error) = os.Lstat

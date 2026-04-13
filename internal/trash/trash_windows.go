//go:build windows

package trash

// Windows trash is implemented via SHFileOperationW with FOF_ALLOWUNDO in
// Sprint 5 ("Cross-platform polish"). Until then we force the fallback
// archive by returning nil from newNative so New() selects the fallback.
func newNative() Trasher { return nil }

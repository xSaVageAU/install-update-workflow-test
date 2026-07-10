//go:build !windows

package uninstall

import "os"

// removeBinary deletes the running executable. Unlike Windows, Unix allows
// unlinking a file that's still mapped/executing: the directory entry goes
// away immediately, and the space is freed once the last open handle (this
// process's own) closes.
func removeBinary(path string) error {
	return os.Remove(path)
}

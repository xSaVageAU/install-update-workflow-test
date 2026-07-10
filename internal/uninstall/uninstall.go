// Package uninstall removes the installed iuw binary and, best-effort, the
// PATH changes the install scripts made for it.
package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
)

// Result describes what Run cleaned up.
type Result struct {
	// Dir is the binary's install directory (its former parent directory).
	Dir string
	// PathRemoved reports whether Dir was found and removed from PATH.
	PathRemoved bool
	// PathWarning is set if cleaning up PATH was attempted but failed.
	// It does not mean Run itself failed: the binary is still gone.
	PathWarning error
}

// Run deletes the currently running executable from disk, then tries to
// remove its directory from PATH the same way install.sh/install.ps1 added
// it. A PATH cleanup failure is reported in the result rather than as an
// error, since the binary has already been removed by that point.
func Run() (Result, error) {
	exePath, err := os.Executable()
	if err != nil {
		return Result{}, fmt.Errorf("locating installed binary: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return Result{}, fmt.Errorf("resolving installed binary path: %w", err)
	}

	if err := removeBinary(exePath); err != nil {
		return Result{}, err
	}

	dir := filepath.Dir(exePath)
	removed, pathErr := removeFromPath(dir)
	return Result{Dir: dir, PathRemoved: removed, PathWarning: pathErr}, nil
}

//go:build !windows

package uninstall

import (
	"bytes"
	"os"
	"path/filepath"
)

// removeFromPath undoes the PATH edit install.sh makes: it looks for the
// exact line install.sh appends to ~/.bashrc or ~/.zshrc and removes it if
// present. It reports whether anything was changed.
func removeFromPath(dir string) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	needle := []byte("\nexport PATH=\"" + dir + ":$PATH\"\n")
	var changed bool
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return changed, err
		}
		if !bytes.Contains(content, needle) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return changed, err
		}
		updated := bytes.Replace(content, needle, []byte("\n"), 1)
		if err := os.WriteFile(path, updated, info.Mode()); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

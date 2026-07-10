//go:build windows

package uninstall

import (
	"fmt"
	"os/exec"
	"strings"
)

// removeFromPath undoes the PATH edit install.ps1 makes: it removes dir
// from the current user's Path environment variable, if present.
//
// The entry-stripping logic (stripPathEntry) is pure so it can be unit
// tested; only reading and writing the actual User PATH value shells out
// to PowerShell, the same tool install.ps1 itself uses, so the change goes
// through .NET's normal environment-broadcast machinery instead of a raw
// registry edit.
func removeFromPath(dir string) (bool, error) {
	current, err := psGetUserPath()
	if err != nil {
		return false, err
	}
	if current == "" {
		return false, nil
	}

	updated, changed := stripPathEntry(current, dir)
	if !changed {
		return false, nil
	}
	if err := psSetUserPath(updated); err != nil {
		return false, err
	}
	return true, nil
}

// stripPathEntry removes dir from a ";"-joined PATH string, comparing
// entries case-insensitively and ignoring a trailing backslash, since
// Windows paths are case-insensitive and either form is valid.
func stripPathEntry(pathVar, dir string) (string, bool) {
	dir = strings.TrimRight(dir, `\`)
	parts := strings.Split(pathVar, ";")
	kept := make([]string, 0, len(parts))
	changed := false
	for _, p := range parts {
		if p != "" && strings.EqualFold(strings.TrimRight(p, `\`), dir) {
			changed = true
			continue
		}
		kept = append(kept, p)
	}
	if !changed {
		return pathVar, false
	}
	return strings.Join(kept, ";"), true
}

func psGetUserPath() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"[Environment]::GetEnvironmentVariable('Path', 'User')").Output()
	if err != nil {
		return "", fmt.Errorf("reading user PATH: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func psSetUserPath(value string) error {
	script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, 'User')", quotePS(value))
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("writing user PATH: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func quotePS(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

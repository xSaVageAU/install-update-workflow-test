//go:build !windows

package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFromPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rc := filepath.Join(home, ".bashrc")
	before := "# existing config\nalias ll='ls -la'\n" +
		"\nexport PATH=\"/home/me/.local/bin:$PATH\"\n"
	if err := os.WriteFile(rc, []byte(before), 0o644); err != nil {
		t.Fatalf("seeding .bashrc: %v", err)
	}

	changed, err := removeFromPath("/home/me/.local/bin")
	if err != nil {
		t.Fatalf("removeFromPath: %v", err)
	}
	if !changed {
		t.Fatal("expected removeFromPath to report a change")
	}

	got, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("reading .bashrc: %v", err)
	}
	want := "# existing config\nalias ll='ls -la'\n"
	if string(got) != want {
		t.Fatalf(".bashrc = %q, want %q", got, want)
	}

	changed, err = removeFromPath("/home/me/.local/bin")
	if err != nil {
		t.Fatalf("removeFromPath (second call): %v", err)
	}
	if changed {
		t.Fatal("expected no further change once the line is gone")
	}
}

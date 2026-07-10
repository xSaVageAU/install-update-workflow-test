package uninstall

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoveBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iuw-test-binary")
	if err := os.WriteFile(path, []byte("binary contents"), 0o755); err != nil {
		t.Fatalf("seeding binary: %v", err)
	}

	if err := removeBinary(path); err != nil {
		t.Fatalf("removeBinary: %v", err)
	}

	// On Windows the actual delete happens asynchronously in a detached
	// helper process, so poll briefly instead of asserting immediately.
	renamed := path + ".uninstall"
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, pathErr := os.Stat(path)
		_, renamedErr := os.Stat(renamed)
		if os.IsNotExist(pathErr) && os.IsNotExist(renamedErr) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %s to be fully removed (path err=%v, renamed err=%v)", path, pathErr, renamedErr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v1.0.0", "v1.1.0", true},
		{"1.0.0", "v1.1.0", true}, // tolerate missing "v" on current
		{"v1.1.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"dev", "v1.0.0", false}, // "dev" builds never look outdated
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestFindAsset(t *testing.T) {
	rel := &Release{Assets: []Asset{
		{Name: "iuw_linux_amd64"},
		{Name: "iuw_darwin_arm64"},
		{Name: "iuw_windows_amd64.exe"},
	}}

	a, err := rel.FindAsset("windows", "amd64")
	if err != nil || a.Name != "iuw_windows_amd64.exe" {
		t.Fatalf("FindAsset(windows, amd64) = %v, %v", a, err)
	}

	a, err = rel.FindAsset("linux", "amd64")
	if err != nil || a.Name != "iuw_linux_amd64" {
		t.Fatalf("FindAsset(linux, amd64) = %v, %v", a, err)
	}

	if _, err := rel.FindAsset("plan9", "amd64"); err == nil {
		t.Fatal("expected error for unsupported platform, got nil")
	}
}

func TestChecksumsAsset(t *testing.T) {
	rel := &Release{Assets: []Asset{{Name: "iuw_linux_amd64"}, {Name: "checksums.txt"}}}
	if _, ok := rel.ChecksumsAsset(); !ok {
		t.Fatal("expected checksums.txt to be found")
	}
	rel = &Release{Assets: []Asset{{Name: "iuw_linux_amd64"}}}
	if _, ok := rel.ChecksumsAsset(); ok {
		t.Fatal("expected no checksums asset")
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.2.3","html_url":"https://example.invalid/v1.2.3","assets":[{"name":"iuw_linux_amd64","browser_download_url":"https://example.invalid/iuw_linux_amd64"}]}`)
	}))
	defer srv.Close()

	oldBase := apiBase
	apiBase = srv.URL
	defer func() { apiBase = oldBase }()

	rel, err := LatestRelease(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v1.2.3" {
		t.Fatalf("TagName = %q, want v1.2.3", rel.TagName)
	}
}

func TestLatestRelease_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oldBase := apiBase
	apiBase = srv.URL
	defer func() { apiBase = oldBase }()

	if _, err := LatestRelease(context.Background(), "owner", "repo"); err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestApplyAt(t *testing.T) {
	newContent := []byte("new binary contents")
	sum := sha256.Sum256(newContent)
	checksums := hex.EncodeToString(sum[:]) + "  iuw_test_asset\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) { w.Write(newContent) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, checksums) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &Release{Assets: []Asset{{Name: "checksums.txt", BrowserDownloadURL: srv.URL + "/checksums.txt"}}}
	asset := &Asset{Name: "iuw_test_asset", BrowserDownloadURL: srv.URL + "/asset"}

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "iuw")
	if err := os.WriteFile(currentPath, []byte("old binary contents"), 0o755); err != nil {
		t.Fatalf("seeding current binary: %v", err)
	}

	if err := applyAt(context.Background(), rel, asset, currentPath); err != nil {
		t.Fatalf("applyAt: %v", err)
	}

	got, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("reading updated binary: %v", err)
	}
	if string(got) != string(newContent) {
		t.Fatalf("binary content = %q, want %q", got, newContent)
	}
	if _, err := os.Stat(currentPath + ".old"); !os.IsNotExist(err) {
		t.Fatalf("expected .old backup to be cleaned up, stat err = %v", err)
	}
}

func TestApplyAt_ChecksumMismatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("new binary contents")) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "0000000000000000000000000000000000000000000000000000000000000000  iuw_test_asset\n")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &Release{Assets: []Asset{{Name: "checksums.txt", BrowserDownloadURL: srv.URL + "/checksums.txt"}}}
	asset := &Asset{Name: "iuw_test_asset", BrowserDownloadURL: srv.URL + "/asset"}

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "iuw")
	original := []byte("old binary contents")
	if err := os.WriteFile(currentPath, original, 0o755); err != nil {
		t.Fatalf("seeding current binary: %v", err)
	}

	if err := applyAt(context.Background(), rel, asset, currentPath); err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	got, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("reading binary after failed update: %v", err)
	}
	if string(got) != string(original) {
		t.Fatal("current binary should be untouched after a checksum failure")
	}
}

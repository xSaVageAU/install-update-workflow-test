package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Apply downloads the given asset, verifies its checksum against the
// release's checksums.txt (when available), and atomically replaces the
// currently running executable with it.
//
// Windows cannot overwrite or delete an executable file while it is mapped
// into a running process, but it can rename one. So the swap goes: rename
// the running binary aside, move the new one into its place, then
// best-effort delete the old one. This mirrors what tools like rustup and
// scoop do for self-update on Windows.
func Apply(ctx context.Context, rel *Release, asset *Asset) error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating running executable: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	return applyAt(ctx, rel, asset, currentPath)
}

// applyAt does the actual download/verify/swap against an explicit target
// path, so the rename dance can be exercised in tests against a scratch
// file instead of the real running test binary.
func applyAt(ctx context.Context, rel *Release, asset *Asset, currentPath string) (err error) {
	dir := filepath.Dir(currentPath)
	downloadedPath := filepath.Join(dir, asset.Name+".new")

	if err := downloadFile(ctx, asset.BrowserDownloadURL, downloadedPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(downloadedPath)
		}
	}()

	if sums, ok := rel.ChecksumsAsset(); ok {
		if err := verifyChecksum(ctx, sums.BrowserDownloadURL, asset.Name, downloadedPath); err != nil {
			return err
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(downloadedPath, 0o755); err != nil {
			return fmt.Errorf("marking new binary executable: %w", err)
		}
	}

	oldPath := currentPath + ".old"
	os.Remove(oldPath) // clean up any leftover from a previous update attempt

	if err := os.Rename(currentPath, oldPath); err != nil {
		return fmt.Errorf("moving current binary aside: %w", err)
	}
	if err := os.Rename(downloadedPath, currentPath); err != nil {
		_ = os.Rename(oldPath, currentPath) // best-effort restore so the command still works
		return fmt.Errorf("installing new binary: %w", err)
	}

	_ = os.Remove(oldPath) // best-effort; a harmless leftover file if this fails

	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", filepath.Base(dest), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: server returned %s", filepath.Base(dest), resp.Status)
	}

	out, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".tmp*")
	if err != nil {
		return err
	}
	tmpPath := out.Name()

	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("saving download: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}
	return os.Rename(tmpPath, dest)
}

func verifyChecksum(ctx context.Context, checksumsURL, assetName, filePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading checksums: server returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	var want string
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum entry found for %s", assetName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing downloaded file: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", assetName, got, want)
	}
	return nil
}

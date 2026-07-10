// Package update checks GitHub Releases for newer versions of this tool
// and applies them by replacing the running binary in place.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// apiBase is a var (not const) so tests can point it at an httptest server.
var apiBase = "https://api.github.com"

// httpClient is shared by every request this package makes. It has no
// Timeout of its own: callers set a deadline on the context they pass in
// (see cmd/iuw's use of context.WithTimeout), so there's a single place
// that controls how long an operation is allowed to take.
var httpClient = &http.Client{}

// Release is the subset of the GitHub releases API response we care about.
type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset is a single downloadable file attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// FindAsset returns the asset whose name matches the given OS/arch. GoReleaser
// is configured (see .goreleaser.yaml) to publish raw binaries rather than
// archives, named "<binary>_<os>_<arch>" with a ".exe" suffix on Windows, so
// the updater never needs to extract a tarball/zip before it can use them.
func (r *Release) FindAsset(goos, goarch string) (*Asset, error) {
	suffix := fmt.Sprintf("_%s_%s", goos, goarch)
	if goos == "windows" {
		suffix += ".exe"
	}
	for i := range r.Assets {
		if strings.HasSuffix(r.Assets[i].Name, suffix) {
			return &r.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("no release asset found for %s/%s", goos, goarch)
}

// ChecksumsAsset returns the checksums.txt asset produced by GoReleaser, if present.
func (r *Release) ChecksumsAsset() (*Asset, bool) {
	for i := range r.Assets {
		if r.Assets[i].Name == "checksums.txt" {
			return &r.Assets[i], true
		}
	}
	return nil, false
}

// LatestRelease fetches the newest published (non-draft, non-prerelease) release.
func LatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s/%s", owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding release info: %w", err)
	}
	return &rel, nil
}

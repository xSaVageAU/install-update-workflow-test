// Package version holds build-time metadata injected via -ldflags.
package version

// These are overwritten at build time, e.g.:
//
//	go build -ldflags "-X github.com/xSaVageAU/install-update-workflow-test/internal/version.Version=v1.2.3"
//
// GoReleaser sets all three automatically (see .goreleaser.yaml).
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

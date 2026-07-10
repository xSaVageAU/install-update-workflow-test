// Command iuw is a small CLI used to experiment with self-update and
// one-command install patterns for Go CLIs.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/xSaVageAU/install-update-workflow-test/internal/update"
	"github.com/xSaVageAU/install-update-workflow-test/internal/version"
)

const (
	repoOwner = "xSaVageAU"
	repoName  = "install-update-workflow-test"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("iuw %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
	case "about":
		runAbout()
	case "update":
		runUpdate(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`iuw - install-update-workflow-test

Usage:
  iuw version        print the running version
  iuw about           print version, commit, build date, and repo
  iuw update          check for a newer release and offer to install it
  iuw update -check   only check for an update, don't install it
  iuw update -yes     install without prompting for confirmation
  iuw help            show this message`)
}

func runAbout() {
	fmt.Printf("Version: %s\nCommit:  %s\nBuilt:   %s\nRepo:    github.com/%s/%s\n",
		version.Version, version.Commit, version.Date, repoOwner, repoName)
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "only check for an update, don't install it")
	yes := fs.Bool("yes", false, "install without prompting for confirmation")
	fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("Checking for updates...")
	rel, err := update.LatestRelease(ctx, repoOwner, repoName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if !update.IsNewer(version.Version, rel.TagName) {
		fmt.Printf("Already up to date (%s).\n", version.Version)
		return
	}

	fmt.Printf("A new version is available: %s -> %s\n", version.Version, rel.TagName)
	if *checkOnly {
		fmt.Println("Run 'iuw update' to install it.")
		return
	}

	if !*yes && !confirm(fmt.Sprintf("Install %s now?", rel.TagName)) {
		fmt.Println("Not updating.")
		return
	}

	asset, err := rel.FindAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("Downloading and installing %s...\n", rel.TagName)
	applyCtx, applyCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer applyCancel()
	if err := update.Apply(applyCtx, rel, asset); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println("Update installed. Run iuw again to use the new version.")
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	var resp string
	fmt.Scanln(&resp)
	resp = strings.ToLower(strings.TrimSpace(resp))
	return resp == "y" || resp == "yes"
}

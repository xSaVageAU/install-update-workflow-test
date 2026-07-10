// Command iuw is a small TUI used to experiment with self-update and
// one-command install patterns for Go CLIs.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xSaVageAU/install-update-workflow-test/internal/tui"
	"github.com/xSaVageAU/install-update-workflow-test/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("iuw %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		return
	}

	p := tea.NewProgram(tui.New())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

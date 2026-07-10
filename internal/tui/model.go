// Package tui implements the application's interactive menu, including the
// "check for updates" / "apply update" flow built on top of internal/update.
package tui

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/xSaVageAU/install-update-workflow-test/internal/update"
	"github.com/xSaVageAU/install-update-workflow-test/internal/version"
)

// repoOwner/repoName identify where this tool's own releases are published.
const (
	repoOwner = "xSaVageAU"
	repoName  = "install-update-workflow-test"
)

type screen int

const (
	screenMenu screen = iota
	screenAbout
	screenChecking
	screenUpToDate
	screenUpdateAvailable
	screenApplying
	screenApplied
	screenError
)

type releaseCheckedMsg struct {
	release *update.Release
	err     error
}

type updateAppliedMsg struct {
	err error
}

// Model is the root Bubble Tea model for the application.
type Model struct {
	screen  screen
	cursor  int
	choices []string
	spinner spinner.Model

	release *update.Release
	asset   *update.Asset
	err     error
}

// New builds the initial model, showing the main menu.
func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		screen:  screenMenu,
		choices: []string{"Check for updates", "About", "Quit"},
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case releaseCheckedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenError
			return m, nil
		}
		m.release = msg.release
		if !update.IsNewer(version.Version, msg.release.TagName) {
			m.screen = screenUpToDate
			return m, nil
		}
		asset, err := msg.release.FindAsset(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			m.err = err
			m.screen = screenError
			return m, nil
		}
		m.asset = asset
		m.screen = screenUpdateAvailable
		return m, nil

	case updateAppliedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenError
			return m, nil
		}
		m.screen = screenApplied
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.screen == screenMenu {
			return m, tea.Quit
		}
	case "esc":
		if m.screen != screenMenu && m.screen != screenChecking && m.screen != screenApplying {
			m.screen = screenMenu
			m.err = nil
			return m, nil
		}
	}

	switch m.screen {
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenUpdateAvailable:
		return m.handleUpdateAvailableKey(msg)
	case screenUpToDate, screenApplied, screenError, screenAbout:
		if msg.String() == "enter" {
			m.screen = screenMenu
			m.err = nil
		}
	}
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter":
		switch m.choices[m.cursor] {
		case "Check for updates":
			m.screen = screenChecking
			return m, tea.Batch(m.spinner.Tick, checkForUpdate())
		case "About":
			m.screen = screenAbout
		case "Quit":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleUpdateAvailableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.screen = screenApplying
		return m, tea.Batch(m.spinner.Tick, applyUpdate(m.release, m.asset))
	case "n":
		m.screen = screenMenu
	}
	return m, nil
}

func checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rel, err := update.LatestRelease(ctx, repoOwner, repoName)
		return releaseCheckedMsg{release: rel, err: err}
	}
}

func applyUpdate(rel *update.Release, asset *update.Asset) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err := update.Apply(ctx, rel, asset)
		return updateAppliedMsg{err: err}
	}
}

func (m Model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenAbout:
		return m.viewAbout()
	case screenChecking:
		return fmt.Sprintf("%s\n\n%s Checking for updates...\n", m.viewTitle(), m.spinner.View())
	case screenUpToDate:
		return fmt.Sprintf("%s\n\n%s\n\n%s", m.viewTitle(),
			successStyle.Render(fmt.Sprintf("You're up to date (%s).", version.Version)),
			helpStyle.Render("enter: back to menu"))
	case screenUpdateAvailable:
		return m.viewUpdateAvailable()
	case screenApplying:
		return fmt.Sprintf("%s\n\n%s Downloading and installing %s...\n", m.viewTitle(), m.spinner.View(), m.release.TagName)
	case screenApplied:
		return fmt.Sprintf("%s\n\n%s\n\n%s", m.viewTitle(),
			successStyle.Render("Update installed. Restart the app to use the new version."),
			helpStyle.Render("enter: back to menu"))
	case screenError:
		return fmt.Sprintf("%s\n\n%s\n\n%s", m.viewTitle(),
			errorStyle.Render("Error: "+m.err.Error()),
			helpStyle.Render("enter: back to menu"))
	}
	return ""
}

func (m Model) viewTitle() string {
	return titleStyle.Render("install-update-workflow-test")
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(m.viewTitle())
	b.WriteString("\n\n")
	for i, choice := range m.choices {
		cursor := "  "
		line := choice
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
			line = selectedItemStyle.Render(choice)
		}
		b.WriteString(cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("version %s  ·  ↑/↓ move · enter select · q quit", version.Version)))
	return b.String()
}

func (m Model) viewAbout() string {
	return fmt.Sprintf(
		"%s\n\nVersion: %s\nCommit:  %s\nBuilt:   %s\nRepo:    github.com/%s/%s\n\n%s",
		m.viewTitle(), version.Version, version.Commit, version.Date, repoOwner, repoName,
		helpStyle.Render("enter: back to menu"),
	)
}

func (m Model) viewUpdateAvailable() string {
	return fmt.Sprintf(
		"%s\n\nA new version is available: %s -> %s\n\n%s",
		m.viewTitle(), version.Version, m.release.TagName,
		helpStyle.Render("y: install now · n: not now"),
	)
}

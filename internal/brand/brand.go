// Package brand holds loom's visual identity — the launch banner (logo,
// tagline, version) — so the logo lives in one place and every entry
// point renders it identically.
package brand

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Version is the current release, shown under the logo.
const Version = "1.0.0"

var (
	artStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	subStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	ruleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

const art = `██╗      ██████╗  ██████╗ ███╗   ███╗
██║     ██╔═══██╗██╔═══██╗████╗ ████║
██║     ██║   ██║██║   ██║██╔████╔██║
██║     ██║   ██║██║   ██║██║╚██╔╝██║
███████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
╚══════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝`

// Banner renders the logo, tagline, and version for a professional launch
// header, e.g. before the wizard, the menu, or help.
func Banner() string {
	sub := subStyle.Render("Project Setup Wizard  ·  AGENTS.md kit  ·  LLM-assisted questions  ·  v" + Version)
	rule := ruleStyle.Render(strings.Repeat("─", 48))
	return artStyle.Render(art) + "\n" + sub + "\n" + rule + "\n"
}

// Small renders a one-line brand mark for quiet commands (status, build,
// update) that don't warrant the full banner.
func Small() string {
	return fmt.Sprintf("%s %s",
		artStyle.Render("loom"),
		subStyle.Render("v"+Version),
	)
}

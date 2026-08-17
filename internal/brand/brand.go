// Package brand holds loom's visual identity — the launch banner (logo,
// tagline, version) — so the logo lives in one place and every entry
// point renders it identically.
package brand

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Version is the current release, shown under the logo.
const Version = "1.0.0"

// Claude-inspired palette: the warm terracotta accent and soft gray
// secondary Claude Code uses, so loom shares its calm, professional look.
const (
	accent = "#D97757" // Claude terracotta
	muted  = "#ADADAD" // soft gray
)

var (
	artStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent))
	subStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(muted))
	ruleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(accent))
)

const art = `██╗      ██████╗  ██████╗ ███╗   ███╗
██║     ██╔═══██╗██╔═══██╗████╗ ████║
██║     ██║   ██║██║   ██║██╔████╔██║
██║     ██║   ██║██║   ██║██║╚██╔╝██║
███████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
╚══════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝`

// ConfigureColor switches the whole UI to the black-and-white (Ascii)
// color profile when color would be lost or unwanted: NO_COLOR is set,
// TERM=dumb, or stdout isn't a terminal (pipes, files, CI logs). Called
// once at startup so the banner, tracer, and TUI all render in clean
// black & white there. In a real terminal the Claude palette is used.
func ConfigureColor() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !stdoutIsTerminal() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

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

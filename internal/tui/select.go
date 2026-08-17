package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	explainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Option mirrors llm.Option, kept local so this UI package doesn't depend
// on the LLM package's shape.
type Option struct {
	Label       string
	Explanation string
}

type selectModel struct {
	prompt      string
	options     []Option
	allowCustom bool
	cursor      int
	customMode  bool
	customInput string
	chosen      string
	explanation string
	quit        bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.customMode {
		switch keyMsg.Type {
		case tea.KeyEnter:
			m.chosen = m.customInput
			m.explanation = ""
			m.quit = true
			return m, tea.Quit
		case tea.KeyEsc:
			if len(m.options) > 0 {
				m.customMode = false
			}
			return m, nil
		case tea.KeyBackspace:
			if len(m.customInput) > 0 {
				m.customInput = m.customInput[:len(m.customInput)-1]
			}
			return m, nil
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyRunes, tea.KeySpace:
			m.customInput += keyMsg.String()
			return m, nil
		default:
			// Anything else (unrecognized escape sequences, function keys,
			// etc.) is ignored rather than blindly stringified into the
			// answer — text input only accepts actual typed characters.
			return m, nil
		}
	}

	switch keyMsg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown:
		max := len(m.options) - 1
		if m.allowCustom {
			max++
		}
		if m.cursor < max {
			m.cursor++
		}
		return m, nil
	case tea.KeyEnter:
		if m.allowCustom && m.cursor == len(m.options) {
			m.customMode = true
			return m, nil
		}
		m.chosen = m.options[m.cursor].Label
		m.explanation = m.options[m.cursor].Explanation
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m selectModel) View() string {
	s := titleStyle.Render(m.prompt) + "\n\n"

	if m.customMode {
		s += "> " + m.customInput + "█\n\n"
		if len(m.options) > 0 {
			s += helpStyle.Render("enter to confirm · esc to go back · ctrl+c to quit")
		} else {
			s += helpStyle.Render("enter to confirm · ctrl+c to quit")
		}
		return s
	}

	for i, opt := range m.options {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		s += fmt.Sprintf("%s%s\n", cursor, opt.Label)
		if m.cursor == i && opt.Explanation != "" {
			s += "    " + explainStyle.Render(opt.Explanation) + "\n"
		}
	}
	if m.allowCustom {
		cursor := "  "
		if m.cursor == len(m.options) {
			cursor = cursorStyle.Render("> ")
		}
		s += fmt.Sprintf("%sType my own answer\n", cursor)
	}
	s += "\n" + helpStyle.Render("↑/↓ to move · enter to choose · ctrl+c to quit")
	return s
}

// RunSelect shows a single-choice question screen and returns the chosen
// label plus the explanation shown for it (empty for a custom answer).
func RunSelect(prompt string, options []Option, allowCustom bool) (label, explanation string, err error) {
	m := selectModel{prompt: prompt, options: options, allowCustom: allowCustom}
	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return "", "", err
	}
	final := res.(selectModel)
	if !final.quit {
		return "", "", fmt.Errorf("cancelled")
	}
	return final.chosen, final.explanation, nil
}

// RunText shows a free-text question screen and returns what was typed.
func RunText(prompt string) (string, error) {
	m := selectModel{prompt: prompt, customMode: true}
	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return "", err
	}
	final := res.(selectModel)
	if !final.quit {
		return "", fmt.Errorf("cancelled")
	}
	return final.chosen, nil
}

package tui

import (
	"strings"

	"github.com/andrinoff/email-cli/config"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loginModel struct {
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
	config     *config.Config
	err        error
}

func initialLoginModel() loginModel {
	m := loginModel{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "gmail or icloud"
			t.Focus()
			t.Prompt = "> "
			t.CharLimit = 10
		case 1:
			t.Placeholder = "your@email.com"
			t.Prompt = "  "
			t.CharLimit = 64
		case 2:
			t.Placeholder = "App-specific password"
			t.Prompt = "  "
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
			t.CharLimit = 64
		}
		m.inputs[i] = t
	}

	return m
}

func (m loginModel) Init() tea.Cmd { return textinput.Blink }

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()
			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.config = &config.Config{
					ServiceProvider: m.inputs[0].Value(),
					Email:           m.inputs[1].Value(),
					Password:        m.inputs[2].Value(),
				}
				if err := config.SaveConfig(m.config); err != nil {
					m.err = err
					return m, nil
				}
				return m, tea.Quit
			}
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].Prompt = "> "
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].Prompt = "  "
			}
			return m, tea.Batch(cmds...)
		}
	}
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *loginModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m loginModel) View() string {
	var b strings.Builder
	b.WriteString(InfoStyle.Render(" Welcome to email-cli! Please configure your email account. ") + "\n\n")
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}
	button := "\n\n" + "[ Submit ]"
	if m.focusIndex == len(m.inputs) {
		button = "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("[ Submit ]")
	}
	b.WriteString(button)
	if m.err != nil {
		b.WriteString("\n\nError: " + m.err.Error())
	}
	b.WriteString(HelpStyle.Render("\n\n(tab to navigate, enter to submit, esc to quit)"))
	return DialogBoxStyle.Render(b.String())
}

// RunLogin starts the login UI and returns the created config.
func RunLogin() (*config.Config, error) {
	p := tea.NewProgram(initialLoginModel())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	finalModel := m.(loginModel)
	return finalModel.config, nil
}
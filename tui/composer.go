package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Composer struct {
	focusIndex int
	inputs     []textinput.Model
	fromAddr   string
}

func NewComposer(from string) Composer {
	m := Composer{
		inputs:   make([]textinput.Model, 3),
		fromAddr: from,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		t.CharLimit = 0 // no limit

		switch i {
		case 0:
			t.Placeholder = "To"
			t.Focus()
			t.Prompt = "> "
		case 1:
			t.Placeholder = "Subject"
			t.Prompt = "> "
		case 2:
			t.Placeholder = "Body..."
			t.Prompt = "> "
		}
		m.inputs[i] = t
	}
	return m
}

func (m Composer) Init() tea.Cmd {
	return textinput.Blink
}

func (m Composer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "tab", "shift+tab", "enter", "up", "down":
			if s == "enter" {
				// If we're on the last input, send the email
				if m.focusIndex == len(m.inputs)-1 {
					return m, func() tea.Msg {
						return SendEmailMsg{
							To:      m.inputs[0].Value(),
							Subject: m.inputs[1].Value(),
							Body:    m.inputs[2].Value(),
						}
					}
				}
			}

			// Cycle focus
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs)-1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Update the focused input
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *Composer) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds = make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Composer) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Compose Email (Press Enter on Body to Send)",
		m.inputs[0].View(),
		m.inputs[1].View(),
		m.inputs[2].View(),
	)
}